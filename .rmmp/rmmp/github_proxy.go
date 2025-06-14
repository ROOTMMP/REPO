package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// GitHub代理API响应结构
type GitHubProxyResponse struct {
	Code       int               `json:"code"`
	Message    string            `json:"msg"`
	Data       []GitHubProxyData `json:"data"`
	Total      int               `json:"total"`
	UpdateTime string            `json:"update_time"`
}

// GitHub代理数据结构
type GitHubProxyData struct {
	URL      string  `json:"url"`
	Server   string  `json:"server"`
	IP       string  `json:"ip"`
	Location string  `json:"location"`
	Latency  int     `json:"latency"`
	Speed    float64 `json:"speed"`
}

// 缓存文件结构
type ProxyCache struct {
	Data       []GitHubProxyData `json:"data"`
	CacheTime  time.Time         `json:"cache_time"`
	UpdateTime string            `json:"update_time"`
	Total      int               `json:"total"`
}

const (
	// GitHub代理API地址
	githubProxyAPI = "https://api.akams.cn/github"
	// 缓存有效期（10小时）
	cacheValidDuration = 10 * time.Hour
)

// getCacheFilePath 获取缓存文件路径，根据平台自动选择
func getCacheFilePath() string {
	if runtime.GOOS == "android" {
		// Android平台使用原路径
		return "/data/adb/modules/rmmp/github_proxys.json"
	} else {
		// 非Android平台使用用户主目录下的路径
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// 如果获取用户主目录失败，使用当前目录
			return "./github_proxys.json"
		}
		return filepath.Join(homeDir, "data", "adb", ".rmm", "CACHE", "github_proxy.json")
	}
}

// GitHubProxyManager GitHub代理管理器
type GitHubProxyManager struct {
	cacheFile string
}

// NewGitHubProxyManager 创建新的GitHub代理管理器
func NewGitHubProxyManager() *GitHubProxyManager {
	return &GitHubProxyManager{
		cacheFile: getCacheFilePath(),
	}
}

// GetProxies 获取GitHub代理列表
func (gpm *GitHubProxyManager) GetProxies() ([]GitHubProxyData, error) {
	// 检查缓存是否有效
	if gpm.isCacheValid() {
		fmt.Println("📦 使用缓存的代理数据")
		return gpm.loadFromCache()
	}

	fmt.Println("🔄 缓存已过期或不存在，正在从API获取最新代理数据...")
	return gpm.fetchFromAPI()
}

// isCacheValid 检查缓存是否有效
func (gpm *GitHubProxyManager) isCacheValid() bool {
	// 检查缓存文件是否存在
	if !fileExists(gpm.cacheFile) {
		return false
	}

	// 读取缓存文件
	cache, err := gpm.readCacheFile()
	if err != nil {
		fmt.Printf("⚠️  读取缓存文件失败: %v\n", err)
		return false
	}

	// 检查缓存时间是否超过10小时
	if time.Since(cache.CacheTime) > cacheValidDuration {
		fmt.Printf("⏰ 缓存已过期 (%.1f小时前更新)\n", time.Since(cache.CacheTime).Hours())
		return false
	}

	fmt.Printf("✅ 缓存有效 (%.1f小时前更新)\n", time.Since(cache.CacheTime).Hours())
	return true
}

// loadFromCache 从缓存加载代理数据
func (gpm *GitHubProxyManager) loadFromCache() ([]GitHubProxyData, error) {
	cache, err := gpm.readCacheFile()
	if err != nil {
		return nil, fmt.Errorf("读取缓存失败: %v", err)
	}

	fmt.Printf("📊 从缓存加载了 %d 个代理地址\n", len(cache.Data))
	return cache.Data, nil
}

// fetchFromAPI 从API获取代理数据
func (gpm *GitHubProxyManager) fetchFromAPI() ([]GitHubProxyData, error) {
	// 发送HTTP请求
	resp, err := http.Get(githubProxyAPI)
	if err != nil {
		return nil, fmt.Errorf("请求API失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析JSON响应
	var apiResponse GitHubProxyResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	// 检查API响应状态
	if apiResponse.Code != 200 {
		return nil, fmt.Errorf("API返回错误: %s", apiResponse.Message)
	}

	fmt.Printf("🌐 从API获取了 %d 个代理地址 (服务器更新时间: %s)\n",
		apiResponse.Total, apiResponse.UpdateTime)

	// 保存到缓存
	if err := gpm.saveToCache(apiResponse); err != nil {
		fmt.Printf("⚠️  保存缓存失败: %v\n", err)
		// 即使保存缓存失败，也返回获取到的数据
	} else {
		fmt.Println("💾 已保存到缓存文件")
	}

	return apiResponse.Data, nil
}

// saveToCache 保存数据到缓存文件
func (gpm *GitHubProxyManager) saveToCache(apiResponse GitHubProxyResponse) error {
	// 创建缓存目录
	cacheDir := filepath.Dir(gpm.cacheFile)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %v", err)
	}

	// 创建缓存数据
	cache := ProxyCache{
		Data:       apiResponse.Data,
		CacheTime:  time.Now(),
		UpdateTime: apiResponse.UpdateTime,
		Total:      apiResponse.Total,
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(gpm.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("写入缓存文件失败: %v", err)
	}

	return nil
}

// readCacheFile 读取缓存文件
func (gpm *GitHubProxyManager) readCacheFile() (*ProxyCache, error) {
	data, err := os.ReadFile(gpm.cacheFile)
	if err != nil {
		return nil, err
	}

	var cache ProxyCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// GetBestProxy 获取最佳代理（延迟最低且速度最快）
func (gpm *GitHubProxyManager) GetBestProxy() (*GitHubProxyData, error) {
	proxies, err := gpm.GetProxies()
	if err != nil {
		return nil, err
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("没有可用的代理")
	}

	// 找到最佳代理（综合考虑延迟和速度）
	var bestProxy *GitHubProxyData
	bestScore := float64(-1)

	for i := range proxies {
		proxy := &proxies[i]
		// 计算综合评分：速度权重0.6，延迟权重0.4（延迟越低越好）
		score := proxy.Speed*0.6 + (1000.0-float64(proxy.Latency))/1000.0*0.4

		if bestScore < 0 || score > bestScore {
			bestScore = score
			bestProxy = proxy
		}
	}

	return bestProxy, nil
}

// ListProxies 列出所有代理并显示详细信息
func (gpm *GitHubProxyManager) ListProxies() error {
	proxies, err := gpm.GetProxies()
	if err != nil {
		return err
	}

	if len(proxies) == 0 {
		fmt.Println("❌ 没有可用的代理")
		return nil
	}

	fmt.Printf("\n📋 GitHub代理列表 (共 %d 个):\n", len(proxies))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-25s %-15s %-15s %-8s %-8s\n", "代理地址", "服务商", "IP地址", "延迟(ms)", "速度(MB/s)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, proxy := range proxies {
		fmt.Printf("%-25s %-15s %-15s %-8d %-8.2f\n",
			proxy.URL, proxy.Server, proxy.IP, proxy.Latency, proxy.Speed)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 显示最佳代理推荐
	bestProxy, err := gpm.GetBestProxy()
	if err == nil {
		fmt.Printf("\n⭐ 推荐代理: %s (延迟: %dms, 速度: %.2fMB/s)\n",
			bestProxy.URL, bestProxy.Latency, bestProxy.Speed)
	}

	return nil
}

// ClearCache 清除缓存文件
func (gpm *GitHubProxyManager) ClearCache() error {
	if !fileExists(gpm.cacheFile) {
		fmt.Println("✅ 缓存文件不存在，无需清除")
		return nil
	}

	if err := os.Remove(gpm.cacheFile); err != nil {
		return fmt.Errorf("删除缓存文件失败: %v", err)
	}

	fmt.Println("🗑️  缓存文件已清除")
	return nil
}

// GetCacheFilePath 获取缓存文件路径
func (gpm *GitHubProxyManager) GetCacheFilePath() string {
	return gpm.cacheFile
}
