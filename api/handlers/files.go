package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/crush/api/models"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// HandleSearchContent 处理搜索文本内容的请求 (参考 OpenCode: /find)
//
//	@Summary		搜索文本内容
//	@Description	在项目中搜索文本内容，支持正则表达式
//	@Tags			File
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			query		query		string	true	"搜索内容"
//	@Param			regex		query		bool	false	"是否使用正则表达式（默认 false）"
//	@Param			case_insensitive	query	bool	false	"是否忽略大小写（默认 false）"
//	@Success		200			{object}	models.SearchResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/find [get]
func (h *Handlers) HandleSearchContent(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	query := string(ctx.Query("query"))
	if query == "" {
		WriteError(c, ctx, "MISSING_QUERY", "Query parameter is required", consts.StatusBadRequest)
		return
	}

	regexParam := ctx.Query("regex")
	caseInsensitiveParam := ctx.Query("case_insensitive")

	useRegex := strings.ToLower(regexParam) == "true" || regexParam == "1"
	caseInsensitive := strings.ToLower(caseInsensitiveParam) == "true" || caseInsensitiveParam == "1"

	// 执行搜索命令
	var results []models.SearchResultItem

	if useRegex {
		// 正则表达式搜索
		cmd := exec.Command("rg", "--no-heading", "--line-number", "--column", query, directory)
		output, err := cmd.Output()
		if err == nil {
			results = h.parseRipgrepOutput(string(output))
		}
	} else {
		// 字面文本搜索
		args := []string{"--fixed-strings", "--no-heading", "--line-number", "--column", query, directory}
		if caseInsensitive {
			args = []string{"--fixed-strings", "--ignore-case", "--no-heading", "--line-number", "--column", query, directory}
		}
		cmd := exec.Command("rg", args...)
		output, err := cmd.Output()
		if err == nil {
			results = h.parseRipgrepOutput(string(output))
		}
	}

	response := models.SearchResponse{
		Query:   query,
		Results: results,
		Count:   len(results),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleSearchFile 处理搜索文件名的请求 (参考 OpenCode: /find/file)
//
//	@Summary		搜索文件名
//	@Description	使用通配符模式搜索文件
//	@Tags			File
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			pattern		query		string	true	"文件名模式（支持通配符）"
//	@Success		200			{object}	models.FileListResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/find/file [get]
func (h *Handlers) HandleSearchFile(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	pattern := string(ctx.Query("pattern"))
	if pattern == "" {
		WriteError(c, ctx, "MISSING_PATTERN", "Pattern parameter is required", consts.StatusBadRequest)
		return
	}

	// 使用 rg 进行文件名搜索
	cmd := exec.Command("rg", "--files", "--glob", pattern, directory)
	output, err := cmd.Output()
	if err != nil {
		// 如果 ripgrep 失败，尝试使用 find 命令
		cmd = exec.Command("find", directory, "-name", pattern)
		output, err = cmd.Output()
		if err != nil {
			response := models.FileListResponse{
				Path:  directory,
				Files: []models.FileInfo{},
				Count: 0,
			}
			WriteJSON(c, ctx, consts.StatusOK, response)
			return
		}
	}

	var files []models.FileInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		info, err := os.Stat(line)
		if err != nil {
			continue
		}

		fileType := "file"
		if info.IsDir() {
			fileType = "directory"
		}

		files = append(files, models.FileInfo{
			Name:    filepath.Base(line),
			Path:    line,
			Type:    fileType,
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	response := models.FileListResponse{
		Path:  directory,
		Files: files,
		Count: len(files),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleListFiles 处理列出目录内容的请求 (参考 OpenCode: /file)
//
//	@Summary		列出目录内容
//	@Description	列出指定目录的文件和子目录
//	@Tags			File
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			path		query		string	false	"子目录路径（相对于项目根目录）"
//	@Param			recursive	query		bool	false	"是否递归列出（默认 false）"
//	@Success		200			{object}	models.FileListResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/file [get]
func (h *Handlers) HandleListFiles(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	path := string(ctx.Query("path"))
	recursiveParam := ctx.Query("recursive")
	recursive := strings.ToLower(recursiveParam) == "true" || recursiveParam == "1"

	// 构建完整路径
	fullPath := directory
	if path != "" {
		fullPath = filepath.Join(directory, path)
	}

	var files []models.FileInfo
	var err error

	if recursive {
		// 递归列出
		files, err = h.listFilesRecursive(fullPath, "")
	} else {
		// 只列出直接子项
		files, err = h.listFilesDirect(fullPath)
	}

	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to list files: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	response := models.FileListResponse{
		Path:  path,
		Files: files,
		Count: len(files),
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleGetFileContent 处理读取文件内容的请求 (参考 OpenCode: /file/content)
//
//	@Summary		读取文件内容
//	@Description	读取指定文件的内容
//	@Tags			File
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Param			path		query		string	true	"文件路径（相对于项目根目录）"
//	@Param			offset		query		int		false	"起始行号（可选）"
//	@Param			limit		query		int		false	"读取行数限制（可选）"
//	@Success		200			{object}	models.FileContentResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/file/content [get]
func (h *Handlers) HandleGetFileContent(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	path := string(ctx.Query("path"))
	if path == "" {
		WriteError(c, ctx, "MISSING_PATH", "Path parameter is required", consts.StatusBadRequest)
		return
	}

	offsetParam := ctx.Query("offset")
	limitParam := ctx.Query("limit")

	offset := 0
	limit := 0

	if offsetParam != "" {
		fmt.Sscanf(offsetParam, "%d", &offset)
	}
	if limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
	}

	// 构建完整路径
	fullPath := filepath.Join(directory, path)

	// 检查文件是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			WriteError(c, ctx, "FILE_NOT_FOUND", "File not found: "+path, consts.StatusNotFound)
			return
		}
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to access file: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		WriteError(c, ctx, "INVALID_PATH", "Path is a directory, not a file", consts.StatusBadRequest)
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(fullPath)
	if err != nil {
		WriteError(c, ctx, "INTERNAL_ERROR", "Failed to read file: "+err.Error(), consts.StatusInternalServerError)
		return
	}

	// 检查是否为二进制文件
	isBinary := isBinaryContent(content)
	encoding := "utf8"
	contentStr := string(content)

	if isBinary {
		// 如果是图片等二进制文件，使用 base64 编码
		if isImageFile(fullPath) {
			encoding = "base64"
			contentStr = base64.StdEncoding.EncodeToString(content)
		} else {
			// 其他二进制文件，截取前 100 字节作为预览
			if len(content) > 100 {
				contentStr = string(content[:100]) + "... (binary file)"
			} else {
				contentStr = string(content)
			}
		}
	} else {
		// 文本文件，处理行号和限制
		lines := strings.Split(contentStr, "\n")
		totalLines := len(lines)

		if offset > 0 || limit > 0 {
			start := offset
			if start < 0 {
				start = 0
			}
			if start >= totalLines {
				lines = []string{}
			} else {
				end := totalLines
				if limit > 0 && start+limit < totalLines {
					end = start + limit
				}
				lines = lines[start:end]
			}
		}

		contentStr = strings.Join(lines, "\n")
	}

	response := models.FileContentResponse{
		Path:     path,
		Content:  contentStr,
		Encoding: encoding,
		Offset:   offset,
		Limit:    limit,
		IsBinary: isBinary,
	}

	if !isBinary && encoding == "utf8" {
		response.Lines = strings.Count(contentStr, "\n") + 1
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// HandleGetGitStatus 处理获取 Git 状态的请求 (参考 OpenCode: /file/status)
//
//	@Summary		获取 Git 状态
//	@Description	获取项目的 Git 状态信息
//	@Tags			File
//	@Accept			json
//	@Produce		json
//	@Param			directory	query		string	true	"项目路径"
//	@Success		200			{object}	models.GitStatusResponse
//	@Failure		400			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]interface{}
//	@Router			/file/status [get]
func (h *Handlers) HandleGetGitStatus(c context.Context, ctx *hertzapp.RequestContext) {
	directory := string(ctx.Query("directory"))
	if directory == "" {
		WriteError(c, ctx, "MISSING_DIRECTORY_PARAM", "Directory query parameter is required", consts.StatusBadRequest)
		return
	}

	// 获取当前分支
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = directory
	branch, _ := branchCmd.Output()
	branchStr := strings.TrimSpace(string(branch))

	// 获取 HEAD 提交
	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headCmd.Dir = directory
	head, _ := headCmd.Output()
	headStr := strings.TrimSpace(string(head))

	// 获取暂存区状态
	stagedCmd := exec.Command("git", "diff", "--cached", "--name-status")
	stagedCmd.Dir = directory
	stagedOutput, _ := stagedCmd.Output()
	staged := parseGitStatusOutput(string(stagedOutput))

	// 获取工作区修改
	modifiedCmd := exec.Command("git", "diff", "--name-status")
	modifiedCmd.Dir = directory
	modifiedOutput, _ := modifiedCmd.Output()
	modified := parseGitStatusOutput(string(modifiedOutput))

	// 获取未跟踪文件
	untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	untrackedCmd.Dir = directory
	untrackedOutput, _ := untrackedCmd.Output()
	untracked := parseGitUntrackedOutput(string(untrackedOutput))

	// 检查是否有删除的文件
	deletedCmd := exec.Command("git", "ls-files", "--deleted")
	deletedCmd.Dir = directory
	deletedOutput, _ := deletedCmd.Output()
	deleted := parseGitDeletedOutput(string(deletedOutput))

	isClean := len(staged) == 0 && len(modified) == 0 && len(untracked) == 0 && len(deleted) == 0

	response := models.GitStatusResponse{
		Branch:    branchStr,
		HEAD:      headStr,
		Staged:    staged,
		Modified:  modified,
		Untracked: untracked,
		Deleted:   deleted,
		IsClean:   isClean,
	}

	WriteJSON(c, ctx, consts.StatusOK, response)
}

// 辅助函数

// parseRipgrepOutput 解析 ripgrep 输出
func (h *Handlers) parseRipgrepOutput(output string) []models.SearchResultItem {
	var results []models.SearchResultItem
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// ripgrep 输出格式: file:line:column:content
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}

		result := models.SearchResultItem{
			Path:    parts[0],
			Content: parts[3],
		}

		fmt.Sscanf(parts[1], "%d", &result.Line)
		fmt.Sscanf(parts[2], "%d", &result.MatchStart)

		results = append(results, result)
	}

	return results
}

// listFilesDirect 直接列出目录内容
func (h *Handlers) listFilesDirect(path string) ([]models.FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []models.FileInfo

	for _, entry := range entries {
		info, _ := entry.Info()
		if info == nil {
			continue
		}

		fileType := "file"
		if entry.IsDir() {
			fileType = "directory"
		}

		files = append(files, models.FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Type:    fileType,
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	return files, nil
}

// listFilesRecursive 递归列出目录内容
func (h *Handlers) listFilesRecursive(path, relativePath string) ([]models.FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []models.FileInfo

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(path, name)
		relPath := filepath.Join(relativePath, name)

		info, _ := entry.Info()
		if info == nil {
			continue
		}

		fileType := "file"
		if entry.IsDir() {
			fileType = "directory"
		}

		fileInfo := models.FileInfo{
			Name:    name,
			Path:    relPath,
			Type:    fileType,
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		}

		if entry.IsDir() {
			// 递归获取子目录内容
			children, err := h.listFilesRecursive(fullPath, relPath)
			if err == nil {
				fileInfo.Children = children
			}
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// isBinaryContent 检查内容是否为二进制
func isBinaryContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// 检查前 1000 个字节
	checkLen := 1000
	if len(content) < checkLen {
		checkLen = len(content)
	}

	for i := 0; i < checkLen; i++ {
		b := content[i]
		// 检查是否为空字节或其他控制字符
		if b == 0 || (b < 0x20 && b != 0x09 && b != 0x0A && b != 0x0D) {
			return true
		}
	}

	return false
}

// isImageFile 检查是否为图片文件
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}

	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}

	return false
}

// parseGitStatusOutput 解析 git status 输出
func parseGitStatusOutput(output string) []models.GitFileStatus {
	var statuses []models.GitFileStatus
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// git diff --name-status 输出格式: status path
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parseGitStatusCode(parts[0])
		path := parts[1]

		fileStatus := models.GitFileStatus{
			Path:   path,
			Status: status,
		}

		if status == "renamed" && len(parts) >= 3 {
			fileStatus.OldPath = parts[1]
			fileStatus.Path = parts[2]
		}

		statuses = append(statuses, fileStatus)
	}

	return statuses
}

// parseGitUntrackedOutput 解析未跟踪文件输出
func parseGitUntrackedOutput(output string) []models.GitFileStatus {
	var statuses []models.GitFileStatus
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		statuses = append(statuses, models.GitFileStatus{
			Path:   line,
			Status: "untracked",
		})
	}

	return statuses
}

// parseGitDeletedOutput 解析已删除文件输出
func parseGitDeletedOutput(output string) []models.GitFileStatus {
	var statuses []models.GitFileStatus
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		statuses = append(statuses, models.GitFileStatus{
			Path:   line,
			Status: "deleted",
		})
	}

	return statuses
}

// parseGitStatusCode 解析 Git 状态码
func parseGitStatusCode(code string) string {
	switch code {
	case "M":
		return "modified"
	case "A":
		return "added"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	case "??":
		return "untracked"
	default:
		return "modified"
	}
}
