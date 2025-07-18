package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 消息类型定义
const (
	MsgTypeMeta    = "meta"    // 文件元信息
	MsgTypeChunk   = "chunk"   // 文件分片
	MsgTypeAck     = "ack"     // 分片接收确认
	MsgTypeResume  = "resume"  // 断点续传请求
	MsgTypeProgress = "progress" // 进度反馈
	MsgTypeError   = "error"   // 错误信息
)

// 基础消息结构
type WSMessage struct {
	Type    string          `json:"type"`    // 消息类型
	Payload json.RawMessage `json:"payload"` // 消息内容
}

// 元信息 payload（客户端 -> 服务器）
type MetaPayload struct {
	FileID      string `json:"file_id"`   // 文件唯一标识（如哈希+文件名）
	FileName    string `json:"file_name"` // 文件名
	FileSize    int64  `json:"file_size"` // 总大小（字节）
	ChunkSize   int64  `json:"chunk_size"`// 分片大小（字节）
	TotalChunks int    `json:"total_chunks"`// 总分片数
}

// 分片数据 payload（客户端 -> 服务器）
type ChunkPayload struct {
	FileID    string `json:"file_id"`   // 文件唯一标识
	ChunkIndex int   `json:"chunk_index"`// 分片索引
	Data      []byte `json:"data"`      // 分片二进制数据（Base64编码）
}

// 分片确认 payload（服务器 -> 客户端）
type AckPayload struct {
	FileID    string `json:"file_id"`
	ChunkIndex int   `json:"chunk_index"`
}

// 断点续传响应 payload（服务器 -> 客户端）
type ResumePayload struct {
	FileID       string   `json:"file_id"`
	ReceivedChunks []int `json:"received_chunks"` // 已接收的分片索引
}

// 服务器存储上传状态
type UploadSession struct {
	FileID       string
	FileName     string
	FileSize     int64
	ChunkSize    int64
	TotalChunks  int
	ReceivedChunks map[int]bool // 记录已接收的分片
	TempDir      string        // 临时分片存储目录
	mu           sync.Mutex
}

var (
	uploadSessions = make(map[string]*UploadSession) // FileID -> 会话
	sessionMu      sync.Mutex
)

func main() {
	http.Handle("/s/", http.FileServer(http.Dir("./")))
	// 启动WebSocket服务器
	http.HandleFunc("/upload", handleWebSocket)
	fmt.Println("Server started at ws://localhost:8080/upload")
	http.ListenAndServe(":8080", nil)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许跨域（生产环境需限制）
		},
		HandshakeTimeout: time.Minute*3,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// 处理消息循环
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error:", err)
			break
		}

	// 解析消息
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		sendError(conn, "Invalid message format")
		continue
	}

	// 根据消息类型处理
	switch msg.Type {
	case MsgTypeMeta:
		handleMeta(conn, msg.Payload)
	case MsgTypeChunk:
		handleChunk(conn, msg.Payload)
	case MsgTypeResume:
		handleResume(conn, msg.Payload)
	default:
		sendError(conn, "Unknown message type")
	}

	fmt.Println("complete")
		//return
	}
}

// 处理文件元信息
func handleMeta(conn *websocket.Conn, payload []byte) {
	var meta MetaPayload
	if err := json.Unmarshal(payload, &meta); err != nil {
		sendError(conn, "Invalid meta data")
		return
	}

	// 创建临时目录存储分片
	tempDir := filepath.Join("temp", meta.FileID)
	os.MkdirAll(tempDir, 0755)

	// 初始化会话
	session := &UploadSession{
		FileID:       meta.FileID,
		FileName:     meta.FileName,
		FileSize:     meta.FileSize,
		ChunkSize:    meta.ChunkSize,
		TotalChunks:  meta.TotalChunks,
		ReceivedChunks: make(map[int]bool),
		TempDir:      tempDir,
	}

	sessionMu.Lock()
	uploadSessions[meta.FileID] = session
	sessionMu.Unlock()

	// 回复确认
	sendMsg(conn, MsgTypeAck, map[string]string{"file_id": meta.FileID})
}

// 处理分片数据
func handleChunk(conn *websocket.Conn, payload []byte) {
	var chunk ChunkPayload
	if err := json.Unmarshal(payload, &chunk); err != nil {
		sendError(conn, "Invalid chunk data")
		return
	}

	sessionMu.Lock()
	session, exists := uploadSessions[chunk.FileID]
	sessionMu.Unlock()
	if !exists {
		sendError(conn, "File session not found")
		return
	}

	// 校验分片索引
	if chunk.ChunkIndex < 0 || chunk.ChunkIndex >= session.TotalChunks {
		sendError(conn, "Invalid chunk index")
		return
	}

	// 存储分片到临时文件
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.ReceivedChunks[chunk.ChunkIndex] {
		// 已接收过，直接确认
		sendAck(conn, chunk.FileID, chunk.ChunkIndex)
		return
	}

	// 写入临时文件（如 temp/xxx/1.dat）
	chunkPath := filepath.Join(session.TempDir, fmt.Sprintf("%d.dat", chunk.ChunkIndex))
	if err := os.WriteFile(chunkPath, chunk.Data, 0644); err != nil {
		sendError(conn, "Failed to save chunk")
		return
	}

	// 标记为已接收
	session.ReceivedChunks[chunk.ChunkIndex] = true
	sendAck(conn, chunk.FileID, chunk.ChunkIndex)

	// 计算进度并反馈
	progress := float64(len(session.ReceivedChunks)) / float64(session.TotalChunks) * 100
	sendProgress(conn, chunk.FileID, progress)

	// 所有分片接收完成，合并文件
	if len(session.ReceivedChunks) == session.TotalChunks {
		mergeFile(session)
		sendMsg(conn, MsgTypeProgress, map[string]interface{}{
			"file_id": progress,
			"progress": 100.0,
			"message": "File upload complete",
		})
	}
}

// 处理断点续传请求
func handleResume(conn *websocket.Conn, payload []byte) {
	var req map[string]string
	json.Unmarshal(payload, &req)
	fileID := req["file_id"]

	sessionMu.Lock()
	session, exists := uploadSessions[fileID]
	sessionMu.Unlock()
	if !exists {
		sendError(conn, "File session not found")
		return
	}

	// 收集已接收的分片索引
	session.mu.Lock()
	defer session.mu.Unlock()
	received := make([]int, 0, len(session.ReceivedChunks))
	for idx := range session.ReceivedChunks {
		received = append(received, idx)
	}

	sendMsg(conn, MsgTypeResume, ResumePayload{
		FileID:       fileID,
		ReceivedChunks: received,
	})
}

// 合并分片为完整文件
func mergeFile(session *UploadSession) error {
	dstPath := filepath.Join("uploads", session.FileName)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 按索引顺序写入分片
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(session.TempDir, fmt.Sprintf("%d.dat", i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return err
		}
		dstFile.Write(chunkData)
		os.Remove(chunkPath) // 删除临时分片
	}

	os.RemoveAll(session.TempDir) // 删除临时目录
	return nil
}

// 工具函数：发送消息
func sendMsg(conn *websocket.Conn, msgType string, payload interface{}) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Failed to marshal payload:", err)
		return
	}
	data, _ := json.Marshal(WSMessage{
		Type:    msgType,
		Payload: json.RawMessage(payloadData),
	})
	conn.WriteMessage(websocket.TextMessage, data)
}

func sendAck(conn *websocket.Conn, fileID string, chunkIndex int) {
	sendMsg(conn, MsgTypeAck, AckPayload{
		FileID:    fileID,
		ChunkIndex: chunkIndex,
	})
}

func sendProgress(conn *websocket.Conn, fileID string, progress float64) {
	fmt.Println(progress)
	sendMsg(conn, MsgTypeProgress, map[string]interface{}{
		"file_id": fileID,
		"progress": progress,
	})
}

func sendError(conn *websocket.Conn, msg string) {
	sendMsg(conn, MsgTypeError, map[string]string{"message": msg})
}