package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// MediaLogger 是一个专门记录媒体文件访问的日志中间件
// 它会完全替代原有的日志系统

// 初始化日志格式
func init() {
	// 设置日志格式为纯文本，不带颜色
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		DisableTimestamp: true, // 禁用默认时间戳，我们将自己格式化
		FullTimestamp: false,
	})
}

// 支持的媒体文件扩展名
var mediaExtensions = map[string]bool{
	// 图片格式
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".svg":  true,
	".tiff": true,
	".ico":  true,
	".heic": true,
	
	// 视频格式
	".mp4":  true,
	".avi":  true,
	".mkv":  true,
	".mov":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
	".m4v":  true,
	".mpg":  true,
	".mpeg": true,
	".3gp":  true,
	".rm":   true,
	".rmvb": true,
	".ts":   true,
	".m3u8": true,
}

// 要忽略的路径前缀
var ignoredPaths = []string{
	"/assets/",
	"/images/",
	"/favicon.ico",
	"/robots.txt",
	"/ping",
}

// 请求和响应的结构体，用于解析JSON
type fsRequest struct {
	Path string `json:"path"`
}

type fsObject struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type int    `json:"type"`
}

type fsListResponse struct {
	Code    int       `json:"code"`
	Content []fsObject `json:"content"`
}

type fsGetResponse struct {
	Code    int     `json:"code"`
	Data    fsObject `json:"data"`
}

// 获取用户名
func getUserName(c *gin.Context) string {
	// 尝试从上下文中获取用户对象
	userObj, exists := c.Get("user")
	if exists {
		// 检查是否可以转换为*model.User类型
		if user, ok := userObj.(*model.User); ok && user != nil {
			return user.Username
		}
		
		// 尝试从map中获取username
		if userMap, ok := userObj.(map[string]interface{}); ok {
			if username, exists := userMap["username"]; exists {
				if usernameStr, ok := username.(string); ok {
					return usernameStr
				}
			}
		}
	}
	
	// 尝试从Authorization头获取token并解析
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return "已认证用户"
	}
	
	// 如果无法获取用户名，返回未知用户
	return "未知用户"
}

// 格式化日志信息为标准格式
func formatMediaLog(timestamp time.Time, clientIP string, filePath string, username string) string {
	// 格式化为"时间：XXXX年X月X日 访问IP：XXX.XXX.XXX.XXX 访问路径：XXX.mp4 用户：XXX"
	return fmt.Sprintf("时间：%s 访问IP：%s 访问路径：%s 用户：%s", 
		timestamp.Format("2006年1月2日 15:04:05"), 
		clientIP, 
		filePath,
		username)
}

// 输出日志到前台和日志文件
func logMediaAccess(timestamp time.Time, clientIP string, filePath string, username string) {
	logMsg := formatMediaLog(timestamp, clientIP, filePath, username)
	
	// 输出到日志文件 - 使用纯文本格式，不带前缀
	log.Info(logMsg)
	
	// 输出到前台控制台
	fmt.Println(logMsg)
}

// MediaLoggerMiddleware 返回一个只记录媒体文件访问的日志中间件
func MediaLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果是静态资源或其他忽略的路径，直接跳过
		path := c.Request.URL.Path
		for _, prefix := range ignoredPaths {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		// 检查是否是直接访问媒体文件的路径
		if isMediaFilePath(path) {
			// 记录直接访问媒体文件的日志
			c.Next()
			
			clientIP := c.ClientIP()
			username := getUserName(c)
			
			// 使用新的日志格式记录
			logMediaAccess(time.Now(), clientIP, path, username)
			return
		}

		// 检查是否是API调用
		if strings.HasPrefix(path, "/api/") {
			// 如果是 /api/fs/list 或 /api/fs/get，需要特殊处理
			if path == "/api/fs/list" || strings.HasPrefix(path, "/api/fs/list?") {
				handleFSListRequest(c)
				return
			} else if path == "/api/fs/get" || strings.HasPrefix(path, "/api/fs/get?") {
				handleFSGetRequest(c)
				return
			}
			
			// 其他API调用不记录日志
			c.Next()
			return
		}

		// 默认情况下不记录日志
		c.Next()
	}
}

// 处理 /api/fs/list 请求
func handleFSListRequest(c *gin.Context) {
	// 保存请求体
	var requestBody []byte
	if c.Request.Body != nil {
		requestBody, _ = io.ReadAll(c.Request.Body)
		// 恢复请求体，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	}

	// 创建响应体捕获器
	responseWriter := &responseBodyWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
	}
	c.Writer = responseWriter
	
	// 处理请求
	c.Next()
	
	// 检查请求体中是否包含媒体文件路径
	var req fsRequest
	if len(requestBody) > 0 {
		_ = json.Unmarshal(requestBody, &req)
	}

	// 检查响应体中是否包含媒体文件
	responseData := responseWriter.body.Bytes()
	var resp fsListResponse
	if len(responseData) > 0 {
		_ = json.Unmarshal(responseData, &resp)
	}

	// 检查响应中是否包含媒体文件
	hasMediaFile := false
	mediaFiles := []string{}
	
	if resp.Code == 200 && len(resp.Content) > 0 {
		for _, item := range resp.Content {
			if isMediaFileName(item.Name) {
				hasMediaFile = true
				mediaFiles = append(mediaFiles, item.Path+"/"+item.Name)
			}
		}
	}

	// 如果包含媒体文件，记录日志
	if hasMediaFile {
		clientIP := c.ClientIP()
		username := getUserName(c)
		
		// 对每个媒体文件记录一条日志
		for _, mediaPath := range mediaFiles {
			logMediaAccess(time.Now(), clientIP, mediaPath, username)
		}
	}
}

// 处理 /api/fs/get 请求
func handleFSGetRequest(c *gin.Context) {
	// 保存请求体
	var requestBody []byte
	if c.Request.Body != nil {
		requestBody, _ = io.ReadAll(c.Request.Body)
		// 恢复请求体，以便后续处理
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	}

	// 创建响应体捕获器
	responseWriter := &responseBodyWriter{
		ResponseWriter: c.Writer,
		body:           &bytes.Buffer{},
	}
	c.Writer = responseWriter

	// 处理请求
	c.Next()
	
	// 检查请求体中是否包含媒体文件路径
	var req fsRequest
	if len(requestBody) > 0 {
		_ = json.Unmarshal(requestBody, &req)
	}

	// 检查响应体中是否包含媒体文件
	responseData := responseWriter.body.Bytes()
	var resp fsGetResponse
	if len(responseData) > 0 {
		_ = json.Unmarshal(responseData, &resp)
	}

	// 检查响应中是否包含媒体文件
	if resp.Code == 200 && isMediaFileName(resp.Data.Name) {
		clientIP := c.ClientIP()
		mediaPath := resp.Data.Path
		username := getUserName(c)
		
		// 使用新的日志格式记录
		logMediaAccess(time.Now(), clientIP, mediaPath, username)
	}
}

// 检查路径是否为媒体文件
func isMediaFilePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return mediaExtensions[ext]
}

// 检查文件名是否为媒体文件
func isMediaFileName(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return mediaExtensions[ext]
}

// responseBodyWriter 是一个用于捕获响应体的包装器
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 实现 ResponseWriter 接口
func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString 实现 ResponseWriter 接口
func (w *responseBodyWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Status 获取状态码
func (w *responseBodyWriter) Status() int {
	return w.ResponseWriter.Status()
}

// 启用调试模式的日志记录器
func MediaLoggerWithDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录所有请求的开始信息
		path := c.Request.URL.Path
		
		// 获取请求体
		var requestBody []byte
		if c.Request.Body != nil && c.Request.Method != "GET" {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// 恢复请求体，以便后续处理
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		
		// 创建响应体捕获器
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseWriter
		
		// 处理请求
		c.Next()
		
		// 检查是否为媒体文件访问
		isMedia := false
		mediaFilePath := path
		
		// 检查路径
		if isMediaFilePath(path) {
			isMedia = true
		}
		
		// 检查请求体
		if !isMedia && len(requestBody) > 0 {
			var req fsRequest
			if err := json.Unmarshal(requestBody, &req); err == nil && req.Path != "" {
				if strings.Contains(req.Path, ".") {
					ext := strings.ToLower(filepath.Ext(req.Path))
					if mediaExtensions[ext] {
						isMedia = true
						mediaFilePath = req.Path
					}
				}
			}
		}
		
		// 检查响应体
		responseData := responseWriter.body.Bytes()
		if !isMedia && len(responseData) > 0 {
			// 尝试解析为列表响应
			var listResp fsListResponse
			if err := json.Unmarshal(responseData, &listResp); err == nil && listResp.Code == 200 {
				for _, item := range listResp.Content {
					if isMediaFileName(item.Name) {
						isMedia = true
						mediaFilePath = item.Path + "/" + item.Name
						break
					}
				}
			}
			
			// 尝试解析为单文件响应
			if !isMedia {
				var getResp fsGetResponse
				if err := json.Unmarshal(responseData, &getResp); err == nil && getResp.Code == 200 {
					if isMediaFileName(getResp.Data.Name) {
						isMedia = true
						mediaFilePath = getResp.Data.Path
					}
				}
			}
		}
		
		// 记录媒体文件访问日志
		if isMedia {
			clientIP := c.ClientIP()
			username := getUserName(c)
			logMediaAccess(time.Now(), clientIP, mediaFilePath, username)
		}
	}
} 