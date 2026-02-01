package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"charm.land/log/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/colorprofile"
	hyperp "github.com/charmbracelet/crush/internal/agent/hyper"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/event"
	"github.com/charmbracelet/crush/internal/oauth"
	"github.com/charmbracelet/crush/internal/oauth/copilot"
	"github.com/charmbracelet/crush/internal/oauth/hyper"
	"github.com/charmbracelet/crush/internal/projects"
	"github.com/charmbracelet/crush/internal/tui"
	"github.com/charmbracelet/crush/internal/version"
	"github.com/charmbracelet/fang"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/charmtone"
	xstrings "github.com/charmbracelet/x/exp/strings"
	"github.com/charmbracelet/x/term"
	"github.com/invopop/jsonschema"
	"github.com/nxadm/tail"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var heartbit = lipgloss.NewStyle().Foreground(charmtone.Dolly).SetString(`
    ▄▄▄▄▄▄▄▄    ▄▄▄▄▄▄▄▄
  ███████████  ███████████
████████████████████████████
████████████████████████████
██████████▀██████▀██████████
██████████ ██████ ██████████
▀▀██████▄████▄▄████▄██████▀▀
  ████████████████████████
    ████████████████████
       ▀▀██████████▀▀
           ▀▀▀▀▀▀
`)

// copied from cobra:
const defaultVersionTemplate = `{{with .DisplayName}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`

var rootCmd = &cobra.Command{
	Use:   "zorkagent",
	Short: "用于复杂问题解决的 AI 助手",
	Long:  "用于复杂问题解决的 AI 助手，可直接访问终端，支持 API 调用",
	Example: `
# 以交互模式运行
zorkagent

# 启用调试日志运行
zorkagent -d

# 在指定目录中启用调试日志运行
zorkagent -d -c /path/to/project

# 使用自定义数据目录运行
zorkagent -D /path/to/custom/.zorkagent

# 打印版本
zorkagent -v

# 运行单个非交互式提示
zorkagent run "解释 Go 中 context 的用法"

# 以危险模式运行（自动接受所有权限）
zorkagent -y
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := setupAppWithProgressBar(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		event.AppInitialized()

		// Set up the TUI.
		var env uv.Environ = os.Environ()
		ui := tui.New(app)
		ui.QueryVersion = shouldQueryTerminalVersion(env)

		program := tea.NewProgram(
			ui,
			tea.WithEnvironment(env),
			tea.WithContext(cmd.Context()),
			tea.WithFilter(tui.MouseEventFilter)) // Filter mouse events based on focus state
		go app.Subscribe(program)

		if _, err := program.Run(); err != nil {
			event.Error(err)
			slog.Error("TUI run error", "error", err)
			return errors.New("ZorkAgent 崩溃。如果已启用指标，我们已收到通知。如果您想报告此问题，请复制上面的堆栈跟踪并提交 issue 到 https://github.com/zorkai/zorkagent/issues/new?template=bug.yml") //nolint:staticcheck
		}
		return nil
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		event.AppExited()
	},
}

var runCmd = &cobra.Command{
	Use:   "run [提示...]",
	Short: "运行单个非交互式提示",
	Long: `以非交互模式运行单个提示并退出。
提示可以作为参数提供或从标准输入管道获取。`,
	Example: `
# 运行简单提示
zorkagent run 解释 Go 中 context 的用法

# 从标准输入管道获取输入
curl https://www.zork.com.cn | zorkagent run "总结这个网站"

# 从文件读取
zorkagent run "这段代码在做什么？" <<< prrr.go

# 以安静模式运行（隐藏加载动画）
zorkagent run --quiet "为该项目生成 README"
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		quiet, _ := cmd.Flags().GetBool("quiet")
		largeModel, _ := cmd.Flags().GetString("model")
		smallModel, _ := cmd.Flags().GetString("small-model")

		// Cancel on SIGINT or SIGTERM.
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer cancel()

		app, err := setupApp(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		if !app.Config().IsConfigured() {
			return fmt.Errorf("未配置任何提供者 - 请运行 'zorkagent' 以交互方式设置提供者")
		}

		prompt := strings.Join(args, " ")

		prompt, err = MaybePrependStdin(prompt)
		if err != nil {
			slog.Error("Failed to read from stdin", "error", err)
			return err
		}

		if prompt == "" {
			return fmt.Errorf("no prompt provided")
		}

		event.SetNonInteractive(true)
		event.AppInitialized()

		return app.RunNonInteractive(ctx, os.Stdout, prompt, largeModel, smallModel, quiet)
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		event.AppExited()
	},
}

var loginCmd = &cobra.Command{
	Aliases: []string{"auth"},
	Use:     "login [平台]",
	Short:   "登录 ZorkAgent 到平台",
	Long: `登录 ZorkAgent 到指定平台。
平台应作为参数提供。
可用平台包括：hyper, copilot。`,
	Example: `
# 使用 Charm Hyper 认证
zorkagent login

# 使用 GitHub Copilot 认证
zorkagent login copilot
  `,
	ValidArgs: []cobra.Completion{
		"hyper",
		"copilot",
		"github",
		"github-copilot",
	},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := setupAppWithProgressBar(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		provider := "hyper"
		if len(args) > 0 {
			provider = args[0]
		}
		switch provider {
		case "hyper":
			return loginHyper()
		case "copilot", "github", "github-copilot":
			return loginCopilot()
		default:
			return fmt.Errorf("unknown platform: %s", args[0])
		}
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "列出项目目录",
	Long:  "列出已知存在 ZorkAgent 项目数据的目录",
	Example: `
# 以表格形式列出所有项目
zorkagent projects

# 以 JSON 格式输出项目数据
zorkagent projects --json
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")

		projectList, err := projects.List()
		if err != nil {
			return err
		}

		if jsonOutput {
			output := struct {
				Projects []projects.Project `json:"projects"`
			}{Projects: projectList}

			data, err := json.Marshal(output)
			if err != nil {
				return err
			}
			cmd.Println(string(data))
			return nil
		}

		if len(projectList) == 0 {
			cmd.Println("No projects tracked yet.")
			return nil
		}

		if term.IsTerminal(os.Stdout.Fd()) {
			// We're in a TTY: make it fancy.
			t := table.New().
				Border(lipgloss.RoundedBorder()).
				StyleFunc(func(row, col int) lipgloss.Style {
					return lipgloss.NewStyle().Padding(0, 2)
				}).
				Headers("Path", "Data Dir", "Last Accessed")

			for _, p := range projectList {
				t.Row(p.Path, p.DataDir, p.LastAccessed.Local().Format("2006-01-02 15:04"))
			}
			lipgloss.Println(t)
			return nil
		}

		// Not a TTY: plain output
		for _, p := range projectList {
			cmd.Printf("%s\t%s\t%s\n", p.Path, p.DataDir, p.LastAccessed.Format("2006-01-02T15:04:05Z07:00"))
		}
		return nil
	},
}

var dirsCmd = &cobra.Command{
	Use:   "dirs",
	Short: "打印 ZorkAgent 使用的目录",
	Long: `打印 ZorkAgent 存储配置和数据文件的目录。
包括全局配置目录和数据目录。`,
	Example: `
# 打印所有目录
zorkagent dirs

# 仅打印配置目录
zorkagent dirs config

# 仅打印数据目录
zorkagent dirs data
  `,
	Run: func(cmd *cobra.Command, args []string) {
		if term.IsTerminal(os.Stdout.Fd()) {
			// We're in a TTY: make it fancy.
			t := table.New().
				Border(lipgloss.RoundedBorder()).
				StyleFunc(func(row, col int) lipgloss.Style {
					return lipgloss.NewStyle().Padding(0, 2)
				}).
				Row("Config", filepath.Dir(config.GlobalConfig())).
				Row("Data", filepath.Dir(config.GlobalConfigData()))
			lipgloss.Println(t)
			return
		}
		// Not a TTY.
		cmd.Println(filepath.Dir(config.GlobalConfig()))
		cmd.Println(filepath.Dir(config.GlobalConfigData()))
	},
}

var configDirCmd = &cobra.Command{
	Use:   "config",
	Short: "打印 ZorkAgent 使用的配置目录",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(filepath.Dir(config.GlobalConfig()))
	},
}

var dataDirCmd = &cobra.Command{
	Use:   "data",
	Short: "打印 ZorkAgent 使用的数据目录",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(filepath.Dir(config.GlobalConfigData()))
	},
}

const defaultTailLines = 1000

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "查看 zorkagent 日志",
	Long:  `查看由 ZorkAgent 生成的日志。此命令允许您查看日志输出以便调试和监控。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := cmd.Flags().GetString("cwd")
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %v", err)
		}

		dataDir, err := cmd.Flags().GetString("data-dir")
		if err != nil {
			return fmt.Errorf("failed to get data directory: %v", err)
		}

		follow, err := cmd.Flags().GetBool("follow")
		if err != nil {
			return fmt.Errorf("failed to get follow flag: %v", err)
		}

		tailLines, err := cmd.Flags().GetInt("tail")
		if err != nil {
			return fmt.Errorf("failed to get tail flag: %v", err)
		}

		log.SetLevel(log.DebugLevel)
		log.SetOutput(os.Stdout)
		if !term.IsTerminal(os.Stdout.Fd()) {
			log.SetColorProfile(colorprofile.NoTTY)
		}

		cfg, err := config.Load(cwd, dataDir, false)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}
		logsFile := filepath.Join(cfg.Options.DataDirectory, "logs", "zorkagent.log")
		_, err = os.Stat(logsFile)
		if os.IsNotExist(err) {
			log.Warn("Looks like you are not in a zorkagent project. No logs found.")
			return nil
		}

		if follow {
			return followLogs(cmd.Context(), logsFile, tailLines)
		}

		return showLogs(logsFile, tailLines)
	},
}

var schemaCmd = &cobra.Command{
	Use:    "schema",
	Short:  "生成配置的 JSON 模式",
	Long:   "为 zorkagent 配置文件生成 JSON 模式",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		reflector := new(jsonschema.Reflector)
		bts, err := json.MarshalIndent(reflector.Reflect(&config.Config{}), "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal schema: %w", err)
		}
		fmt.Println(string(bts))
		return nil
	},
}

var updateProvidersSource string

var updateProvidersCmd = &cobra.Command{
	Use:   "update-providers [路径或URL]",
	Short: "更新提供者",
	Long:  `从指定的本地路径或远程 URL 更新提供者信息。`,
	Example: `
# 远程更新 Catwalk 提供者（默认）
zorkagent update-providers

# 从自定义 URL 更新 Catwalk 提供者
zorkagent update-providers https://example.com/providers.json

# 从本地文件更新 Catwalk 提供者
zorkagent update-providers /path/to/local-providers.json

# 从嵌入版本更新 Catwalk 提供者
zorkagent update-providers embedded

# 更新 Hyper 提供者信息
zorkagent update-providers --source=hyper

# 从自定义 URL 更新 Hyper
zorkagent update-providers --source=hyper https://hyper.example.com
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// NOTE(@andreynering): We want to skip logging output do stdout here.
		slog.SetDefault(slog.New(slog.DiscardHandler))

		var pathOrURL string
		if len(args) > 0 {
			pathOrURL = args[0]
		}

		var err error
		switch updateProvidersSource {
		case "catwalk":
			err = config.UpdateProviders(pathOrURL)
		case "hyper":
			err = config.UpdateHyper(pathOrURL)
		default:
			return fmt.Errorf("invalid source %q, must be 'catwalk' or 'hyper'", updateProvidersSource)
		}

		if err != nil {
			return err
		}

		// NOTE(@andreynering): This style is more-or-less copied from Fang's
		// error message, adapted for success.
		headerStyle := lipgloss.NewStyle().
			Foreground(charmtone.Butter).
			Background(charmtone.Guac).
			Bold(true).
			Padding(0, 1).
			Margin(1).
			MarginLeft(2).
			SetString("SUCCESS")
		textStyle := lipgloss.NewStyle().
			MarginLeft(2).
			SetString(fmt.Sprintf("%s provider updated successfully.", updateProvidersSource))

		fmt.Printf("%s\n%s\n\n", headerStyle.Render(), textStyle.Render())
		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动无头 API 服务器",
	Long: `启动无头 API 服务器以编程方式访问 ZorkAgent。

服务器提供用于管理项目、会话和消息的 REST API。
当您想将 ZorkAgent 集成到其他应用程序或脚本时使用此功能。`,
	Example: `
# 在默认端口（8080）启动服务器
zorkagent serve

# 在自定义端口启动服务器
zorkagent serve --port 3000

# 在所有接口上启动服务器
zorkagent serve --host 0.0.0.0

# 启用调试日志启动服务器
zorkagent serve --debug
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")
		StartServer(cmd, port, host)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringP("cwd", "c", "", "当前工作目录")
	rootCmd.PersistentFlags().StringP("data-dir", "D", "", "自定义 zorkagent 数据目录")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "调试模式")
	rootCmd.Flags().BoolP("help", "h", false, "帮助")
	rootCmd.Flags().BoolP("yolo", "y", false, "自动接受所有权限（危险模式）")

	runCmd.Flags().BoolP("quiet", "q", false, "隐藏加载动画")
	runCmd.Flags().StringP("model", "m", "", "Model to use. Accepts 'model' or 'provider/model' to disambiguate models with the same name across providers")
	runCmd.Flags().String("small-model", "", "Small model to use. If not provided, uses the default small model for the provider")
	projectsCmd.Flags().Bool("json", false, "以 JSON 格式输出")
	dirsCmd.AddCommand(configDirCmd, dataDirCmd)
	logsCmd.Flags().BoolP("follow", "f", false, "跟踪日志输出")
	logsCmd.Flags().IntP("tail", "t", defaultTailLines, "仅显示最后 N 行，默认：1000 以提高性能")
	updateProvidersCmd.Flags().StringVar(&updateProvidersSource, "source", "catwalk", "要更新的提供者源（catwalk 或 hyper）")
	serveCmd.Flags().IntP("port", "p", 8080, "API 服务器端口")
	serveCmd.Flags().String("host", "localhost", "API 服务器主机")

	rootCmd.AddCommand(
		runCmd,
		dirsCmd,
		projectsCmd,
		updateProvidersCmd,
		logsCmd,
		schemaCmd,
		loginCmd,
		serveCmd,
	)
}

// Execute 是命令入口点
func Execute() {
	// NOTE: very hacky: we create a colorprofile writer with STDOUT, then make
	// it forward to a bytes.Buffer, write the colored heartbit to it, and then
	// finally prepend it in the version template.
	// Unfortunately cobra doesn't give us a way to set a function to handle
	// printing the version, and PreRunE runs after the version is already
	// handled, so that doesn't work either.
	// This is the only way I could find that works relatively well.
	if term.IsTerminal(os.Stdout.Fd()) {
		var b bytes.Buffer
		w := colorprofile.NewWriter(os.Stdout, os.Environ())
		w.Forward = &b
		_, _ = w.WriteString(heartbit.String())
		rootCmd.SetVersionTemplate(b.String() + "\n" + defaultVersionTemplate)
	}
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}

// supportsProgressBar tries to determine whether the current terminal supports
// progress bars by looking into environment variables.
func supportsProgressBar() bool {
	if !term.IsTerminal(os.Stderr.Fd()) {
		return false
	}
	termProg := os.Getenv("TERM_PROGRAM")
	_, isWindowsTerminal := os.LookupEnv("WT_SESSION")

	return isWindowsTerminal || strings.Contains(strings.ToLower(termProg), "ghostty")
}

func setupAppWithProgressBar(cmd *cobra.Command) (*app.App, error) {
	if supportsProgressBar() {
		_, _ = fmt.Fprintf(os.Stderr, ansi.SetIndeterminateProgressBar)
		defer func() { _, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar) }()
	}

	return setupApp(cmd)
}

// setupApp handles the common setup logic for both interactive and non-interactive modes.
// It returns the app instance, config, cleanup function, and any error.
func setupApp(cmd *cobra.Command) (*app.App, error) {
	debug, _ := cmd.Flags().GetBool("debug")
	yolo, _ := cmd.Flags().GetBool("yolo")
	dataDir, _ := cmd.Flags().GetString("data-dir")
	ctx := cmd.Context()

	cwd, err := ResolveCwd(cmd)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Init(cwd, dataDir, debug)
	if err != nil {
		return nil, err
	}

	if cfg.Permissions == nil {
		cfg.Permissions = &config.Permissions{}
	}
	cfg.Permissions.SkipRequests = yolo

	if err := createDotZorkAgentDir(cfg.Options.DataDirectory); err != nil {
		return nil, err
	}

	// Register this project in the centralized projects list.
	if err := projects.Register(cwd, cfg.Options.DataDirectory); err != nil {
		slog.Warn("Failed to register project", "error", err)
		// Non-fatal: continue even if registration fails
	}

	// Connect to DB; this will also run migrations.
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return nil, err
	}

	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		slog.Error("Failed to create app instance", "error", err)
		return nil, err
	}

	if shouldEnableMetrics() {
		event.Init()
	}

	return appInstance, nil
}

func shouldEnableMetrics() bool {
	if v, _ := strconv.ParseBool(os.Getenv("ZORKAGENT_DISABLE_METRICS")); v {
		return false
	}
	if v, _ := strconv.ParseBool(os.Getenv("DO_NOT_TRACK")); v {
		return false
	}
	if config.Get().Options.DisableMetrics {
		return false
	}
	return true
}

func MaybePrependStdin(prompt string) (string, error) {
	if term.IsTerminal(os.Stdin.Fd()) {
		return prompt, nil
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return prompt, err
	}
	// Check if stdin is a named pipe ( | ) or regular file ( < ).
	if fi.Mode()&os.ModeNamedPipe == 0 && !fi.Mode().IsRegular() {
		return prompt, nil
	}
	bts, err := io.ReadAll(os.Stdin)
	if err != nil {
		return prompt, err
	}
	return string(bts) + "\n\n" + prompt, nil
}

func ResolveCwd(cmd *cobra.Command) (string, error) {
	cwd, _ := cmd.Flags().GetString("cwd")
	if cwd != "" {
		err := os.Chdir(cwd)
		if err != nil {
			return "", fmt.Errorf("failed to change directory: %v", err)
		}
		return cwd, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}
	return cwd, nil
}

func createDotZorkAgentDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create data directory: %q %w", dir, err)
	}

	gitIgnorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitIgnorePath, []byte("*\n"), 0o644); err != nil {
			return fmt.Errorf("failed to create .gitignore file: %q %w", gitIgnorePath, err)
		}
	}

	return nil
}

func shouldQueryTerminalVersion(env uv.Environ) bool {
	termType := env.Getenv("TERM")
	termProg, okTermProg := env.LookupEnv("TERM_PROGRAM")
	_, okSSHTTY := env.LookupEnv("SSH_TTY")
	return (!okTermProg && !okSSHTTY) ||
		(!strings.Contains(termProg, "Apple") && !okSSHTTY) ||
		// Terminals that do support XTVERSION.
		xstrings.ContainsAnyOf(termType, "alacritty", "ghostty", "kitty", "rio", "wezterm")
}

func loginHyper() error {
	cfg := config.Get()
	if !hyperp.Enabled() {
		return fmt.Errorf("hyper not enabled")
	}
	ctx := getLoginContext()

	resp, err := hyper.InitiateDeviceAuth(ctx)
	if err != nil {
		return err
	}

	if clipboard.WriteAll(resp.UserCode) == nil {
		fmt.Println("The following code should be on clipboard already:")
	} else {
		fmt.Println("Copy the following code:")
	}

	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Bold(true).Render(resp.UserCode))
	fmt.Println()
	fmt.Println("Press enter to open this URL, and then paste it there:")
	fmt.Println()
	fmt.Println(lipgloss.NewStyle().Hyperlink(resp.VerificationURL, "id=hyper").Render(resp.VerificationURL))
	fmt.Println()
	waitEnter()
	if err := browser.OpenURL(resp.VerificationURL); err != nil {
		fmt.Println("Could not open the URL. You'll need to manually open the URL in your browser.")
	}

	fmt.Println("Exchanging authorization code...")
	refreshToken, err := hyper.PollForToken(ctx, resp.DeviceCode, resp.ExpiresIn)
	if err != nil {
		return err
	}

	fmt.Println("Exchanging refresh token for access token...")
	token, err := hyper.ExchangeToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	fmt.Println("Verifying access token...")
	introspect, err := hyper.IntrospectToken(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("token introspection failed: %w", err)
	}
	if !introspect.Active {
		return fmt.Errorf("access token is not active")
	}

	if err := cmp.Or(
		cfg.SetConfigField("providers.hyper.api_key", token.AccessToken),
		cfg.SetConfigField("providers.hyper.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with Hyper!")
	return nil
}

func loginCopilot() error {
	ctx := getLoginContext()

	cfg := config.Get()
	if cfg.HasConfigField("providers.copilot.oauth") {
		fmt.Println("You are already logged in to GitHub Copilot.")
		return nil
	}

	diskToken, hasDiskToken := copilot.RefreshTokenFromDisk()
	var token *oauth.Token

	switch {
	case hasDiskToken:
		fmt.Println("Found existing GitHub Copilot token on disk. Using it to authenticate...")

		t, err := copilot.RefreshToken(ctx, diskToken)
		if err != nil {
			return fmt.Errorf("unable to refresh token from disk: %w", err)
		}
		token = t
	default:
		fmt.Println("Requesting device code from GitHub...")
		dc, err := copilot.RequestDeviceCode(ctx)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Open the following URL and follow the instructions to authenticate with GitHub Copilot:")
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().Hyperlink(dc.VerificationURI, "id=copilot").Render(dc.VerificationURI))
		fmt.Println()
		fmt.Println("Code:", lipgloss.NewStyle().Bold(true).Render(dc.UserCode))
		fmt.Println()
		fmt.Println("Waiting for authorization...")

		t, err := copilot.PollForToken(ctx, dc)
		if err == copilot.ErrNotAvailable {
			fmt.Println()
			fmt.Println("GitHub Copilot is unavailable for this account. To signup, go to the following page:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.SignupURL, "id=copilot-signup").Render(copilot.SignupURL))
			fmt.Println()
			fmt.Println("You may be able to request free access if eligible. For more information, see:")
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Hyperlink(copilot.FreeURL, "id=copilot-free").Render(copilot.FreeURL))
		}
		if err != nil {
			return err
		}
		token = t
	}

	if err := cmp.Or(
		cfg.SetConfigField("providers.copilot.api_key", token.AccessToken),
		cfg.SetConfigField("providers.copilot.oauth", token),
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're now authenticated with GitHub Copilot!")
	return nil
}

func getLoginContext() context.Context {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	go func() {
		<-ctx.Done()
		cancel()
		os.Exit(1)
	}()
	return ctx
}

func waitEnter() {
	_, _ = fmt.Scanln()
}

func followLogs(ctx context.Context, logsFile string, tailLines int) error {
	t, err := tail.TailFile(logsFile, tail.Config{
		Follow: false,
		ReOpen: false,
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to tail log file: %v", err)
	}

	var lines []string
	for line := range t.Lines {
		if line.Err != nil {
			continue
		}
		lines = append(lines, line.Text)
		if len(lines) > tailLines {
			lines = lines[len(lines)-tailLines:]
		}
	}
	t.Stop()

	for _, line := range lines {
		printLogLine(line)
	}

	if len(lines) == tailLines {
		fmt.Fprintf(os.Stderr, "\nShowing last %d lines. Full logs available at: %s\n", tailLines, logsFile)
		fmt.Fprintf(os.Stderr, "Following new log entries...\n\n")
	}

	t, err = tail.TailFile(logsFile, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Logger:   tail.DiscardingLogger,
		Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
	})
	if err != nil {
		return fmt.Errorf("failed to tail log file: %v", err)
	}
	defer t.Stop()

	for {
		select {
		case line := <-t.Lines:
			if line.Err != nil {
				continue
			}
			printLogLine(line.Text)
		case <-ctx.Done():
			return nil
		}
	}
}

func showLogs(logsFile string, tailLines int) error {
	t, err := tail.TailFile(logsFile, tail.Config{
		Follow:      false,
		ReOpen:      false,
		Logger:      tail.DiscardingLogger,
		MaxLineSize: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to tail log file: %v", err)
	}
	defer t.Stop()

	var lines []string
	for line := range t.Lines {
		if line.Err != nil {
			continue
		}
		lines = append(lines, line.Text)
		if len(lines) > tailLines {
			lines = lines[len(lines)-tailLines:]
		}
	}

	for _, line := range lines {
		printLogLine(line)
	}

	if len(lines) == tailLines {
		fmt.Fprintf(os.Stderr, "\nShowing last %d lines. Full logs available at: %s\n", tailLines, logsFile)
	}

	return nil
}

func printLogLine(lineText string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(lineText), &data); err != nil {
		return
	}
	msg := data["msg"]
	level := data["level"]
	otherData := []any{}
	keys := []string{}
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		switch k {
		case "msg", "level", "time":
			continue
		case "source":
			source, ok := data[k].(map[string]any)
			if !ok {
				continue
			}
			sourceFile := fmt.Sprintf("%s:%d", source["file"], int(source["line"].(float64)))
			otherData = append(otherData, "source", sourceFile)

		default:
			otherData = append(otherData, k, data[k])
		}
	}
	log.SetTimeFunction(func(_ time.Time) time.Time {
		// parse the timestamp from the log line if available
		t, err := time.Parse(time.RFC3339, data["time"].(string))
		if err != nil {
			return time.Now() // fallback to current time if parsing fails
		}
		return t
	})
	switch level {
	case "INFO":
		log.Info(msg, otherData...)
	case "DEBUG":
		log.Debug(msg, otherData...)
	case "ERROR":
		log.Error(msg, otherData...)
	case "WARN":
		log.Warn(msg, otherData...)
	default:
		log.Info(msg, otherData...)
	}
}
