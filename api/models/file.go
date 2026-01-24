package models

// 搜索文本内容请求
type SearchContentRequest struct {
	Query    string `json:"query"`
	Path     string `json:"path,omitempty"`
	Regex    bool   `json:"regex,omitempty"`
	CaseInsensitive bool `json:"case_insensitive,omitempty"`
}

// 搜索结果项
type SearchResultItem struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Content  string `json:"content"`
	MatchStart int  `json:"match_start,omitempty"`
	MatchEnd   int  `json:"match_end,omitempty"`
}

// 搜索响应
type SearchResponse struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
	Count   int                `json:"count"`
}

// 文件搜索请求
type SearchFileRequest struct {
	Pattern  string `json:"pattern"`
	Path     string `json:"path,omitempty"`
}

// 文件信息
type FileInfo struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Type      string      `json:"type"` // "file" or "directory"
	Size      int64       `json:"size,omitempty"`
	Mode      string      `json:"mode,omitempty"`
	ModTime   string      `json:"mod_time,omitempty"`
	Children  []FileInfo  `json:"children,omitempty"`
}

// 文件列表响应
type FileListResponse struct {
	Path     string     `json:"path"`
	Files    []FileInfo `json:"files"`
	Count    int        `json:"count"`
}

// 文件内容请求
type FileContentRequest struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// 文件内容响应
type FileContentResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "utf8", "base64" for images
	Lines    int   `json:"lines,omitempty"` // 行数（如果是文本文件）
	Offset   int   `json:"offset,omitempty"`
	Limit    int   `json:"limit,omitempty"`
	IsBinary bool  `json:"is_binary,omitempty"`
}

// Git 文件状态
type GitFileStatus struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // "modified", "added", "deleted", "renamed", "untracked"
	OldPath   string `json:"old_path,omitempty"` // for renamed files
}

// Git 状态响应
type GitStatusResponse struct {
	Branch       string           `json:"branch"`
	HEAD         string           `json:"head,omitempty"`
	Staged       []GitFileStatus  `json:"staged"`
	Modified     []GitFileStatus  `json:"modified"`
	Untracked    []GitFileStatus  `json:"untracked"`
	Deleted      []GitFileStatus  `json:"deleted"`
	IsClean      bool             `json:"is_clean"`
}

// LSP 状态
type LSPStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Root   string `json:"root"`
	Status string `json:"status"` // "connected" or "error"
}

// MCP 状态（通用格式，使用 map[string]interface{} 来支持多种状态类型）
type MCPStatus map[string]interface{}

// MCP 状态 - 已连接
type MCPStatusConnected struct {
	Status string `json:"status"` // "connected"
}

// MCP 状态 - 已禁用
type MCPStatusDisabled struct {
	Status string `json:"status"` // "disabled"
}

// MCP 状态 - 失败
type MCPStatusFailed struct {
	Status  string `json:"status"` // "failed"
	Message string `json:"message"`
}

// MCP 状态 - 需要认证
type MCPStatusNeedsAuth struct {
	Status string `json:"status"` // "needs_auth"
	URL    string `json:"url"`
}

// MCP 状态 - 需要客户端注册
type MCPStatusNeedsClientRegistration struct {
	Status      string `json:"status"` // "needs_client_registration"
	RedirectURI string `json:"redirect_uri"`
}
