package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// UpdateInfo 表示update.json文件的结构
type UpdateInfo struct {
	Changelog   string `json:"changelog"`
	Version     string `json:"version"`
	VersionCode int    `json:"versionCode"`
	ZipURL      string `json:"zipUrl"`
}

// ModuleDownloader 模块下载器
type ModuleDownloader struct {
	gpm      *GitHubProxyManager
	cacheDir string
	timeout  time.Duration
	maxRetry int
}

// NewModuleDownloader 创建新的模块下载器
func NewModuleDownloader() *ModuleDownloader {
	return &ModuleDownloader{
		gpm:      NewGitHubProxyManager(),
		cacheDir: getDownloadCacheDir(),
		timeout:  3 * time.Second, // API请求超时3秒
		maxRetry: 10,              // 最多尝试10个代理
	}
}

// getDownloadCacheDir 获取下载缓存目录
func getDownloadCacheDir() string {
	if runtime.GOOS == "android" {
		return "/data/adb/modules/rmmp/downloads"
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "./downloads"
		}
		return filepath.Join(homeDir, "data", "adb", ".rmm", "CACHE", "downloads")
	}
}

// normalizeRepoName 规范化仓库名称 (支持 username/repo 和 username\repo)
func (md *ModuleDownloader) normalizeRepoName(repo string) string {
	// 将反斜杠替换为正斜杠
	repo = strings.ReplaceAll(repo, "\\", "/")

	// 确保格式为 username/repo
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return ""
	}

	return fmt.Sprintf("%s/%s", parts[0], parts[1])
}

// buildUpdateURL 构建update.json的下载URL
func (md *ModuleDownloader) buildUpdateURL(repo string) string {
	return fmt.Sprintf("https://github.com/%s/releases/latest/download/update.json", repo)
}

// downloadWithTimeout 带超时的下载函数
func (md *ModuleDownloader) downloadWithTimeout(url string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// downloadUpdateJSON 下载update.json文件
func (md *ModuleDownloader) downloadUpdateJSON(repo string) (*UpdateInfo, error) {
	originalURL := md.buildUpdateURL(repo)
	fmt.Printf("🔄 正在下载 %s 的更新信息...\n", repo)

	// 首先尝试原始链接
	fmt.Printf("📡 尝试原始链接: %s\n", originalURL)
	data, err := md.downloadWithTimeout(originalURL, md.timeout)
	if err == nil {
		fmt.Println("✅ 原始链接下载成功")
		return md.parseUpdateJSON(data)
	}

	fmt.Printf("⚠️  原始链接失败: %v\n", err)
	fmt.Println("🔄 正在尝试代理链接...")

	// 获取代理列表并按速度排序
	proxies, err := md.gpm.GetProxies()
	if err != nil {
		return nil, fmt.Errorf("获取代理列表失败: %v", err)
	}

	// 按速度降序排序
	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].Speed > proxies[j].Speed
	})

	// 尝试每个代理
	tried := 0
	for _, proxy := range proxies {
		if tried >= md.maxRetry {
			break
		}

		proxyURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(proxy.URL, "/"), originalURL)
		fmt.Printf("📡 尝试代理 [%d/%d]: %s (速度: %.2fMB/s, 延迟: %dms)\n",
			tried+1, md.maxRetry, proxy.URL, proxy.Speed, proxy.Latency)

		data, err := md.downloadWithTimeout(proxyURL, md.timeout)
		if err == nil {
			fmt.Printf("✅ 代理下载成功: %s\n", proxy.URL)
			return md.parseUpdateJSON(data)
		}

		fmt.Printf("❌ 代理失败: %v\n", err)
		tried++
	}

	return nil, fmt.Errorf("所有下载尝试均失败")
}

// parseUpdateJSON 解析update.json内容
func (md *ModuleDownloader) parseUpdateJSON(data []byte) (*UpdateInfo, error) {
	var updateInfo UpdateInfo
	if err := json.Unmarshal(data, &updateInfo); err != nil {
		return nil, fmt.Errorf("解析update.json失败: %v", err)
	}

	if updateInfo.ZipURL == "" {
		return nil, fmt.Errorf("update.json中没有找到zipUrl字段")
	}

	return &updateInfo, nil
}

// downloadModule 下载模块zip文件
func (md *ModuleDownloader) downloadModule(updateInfo *UpdateInfo) (string, error) {
	// 创建下载目录
	if err := os.MkdirAll(md.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("创建下载目录失败: %v", err)
	}

	// 生成本地文件名
	fileName := fmt.Sprintf("module_%s_%d.zip",
		strings.ReplaceAll(updateInfo.Version, "/", "_"),
		updateInfo.VersionCode)
	localPath := filepath.Join(md.cacheDir, fileName)

	fmt.Printf("🔄 正在下载模块: %s\n", updateInfo.Version)
	fmt.Printf("📁 保存位置: %s\n", localPath)

	// 首先尝试原始链接
	originalURL := updateInfo.ZipURL
	fmt.Printf("📡 尝试原始链接下载...\n")

	err := md.downloadFile(originalURL, localPath, 30*time.Second) // 模块下载使用30秒超时
	if err == nil {
		fmt.Println("✅ 原始链接下载成功")
		return localPath, nil
	}

	fmt.Printf("⚠️  原始链接下载失败: %v\n", err)

	// 如果原始URL已经包含代理，尝试提取原始GitHub URL
	githubURL := md.extractGitHubURL(originalURL)
	if githubURL != originalURL {
		fmt.Printf("🔄 尝试提取的GitHub原始链接: %s\n", githubURL)
		err = md.downloadFile(githubURL, localPath, 30*time.Second)
		if err == nil {
			fmt.Println("✅ GitHub原始链接下载成功")
			return localPath, nil
		}
		fmt.Printf("⚠️  GitHub原始链接下载失败: %v\n", err)
	}

	// 尝试代理下载
	fmt.Println("🔄 正在尝试代理下载...")
	return md.downloadWithProxies(githubURL, localPath)
}

// extractGitHubURL 从代理URL中提取原始的GitHub URL
func (md *ModuleDownloader) extractGitHubURL(proxyURL string) string {
	// 常见的代理前缀模式
	prefixes := []string{
		"https://ghproxy.cc/",
		"https://ghproxy.cn/",
		"https://gh.b52m.cn/",
		"https://github.moeyy.xyz/",
		"https://mirror.ghproxy.com/",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(proxyURL, prefix) {
			return strings.TrimPrefix(proxyURL, prefix)
		}
	}

	return proxyURL
}

// downloadWithProxies 使用代理下载文件
func (md *ModuleDownloader) downloadWithProxies(originalURL, localPath string) (string, error) {
	proxies, err := md.gpm.GetProxies()
	if err != nil {
		return "", fmt.Errorf("获取代理列表失败: %v", err)
	}

	// 按速度降序排序
	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].Speed > proxies[j].Speed
	})

	tried := 0
	for _, proxy := range proxies {
		if tried >= md.maxRetry {
			break
		}

		proxyURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(proxy.URL, "/"), originalURL)
		fmt.Printf("📡 尝试代理 [%d/%d]: %s\n", tried+1, md.maxRetry, proxy.URL)

		err := md.downloadFile(proxyURL, localPath, 30*time.Second)
		if err == nil {
			fmt.Printf("✅ 代理下载成功: %s\n", proxy.URL)
			return localPath, nil
		}

		fmt.Printf("❌ 代理下载失败: %v\n", err)
		tried++
	}

	return "", fmt.Errorf("所有代理下载尝试均失败")
}

// downloadFile 下载文件到本地
func (md *ModuleDownloader) downloadFile(url, localPath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 创建本地文件
	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 复制文件内容
	_, err = io.Copy(file, resp.Body)
	return err
}

// confirmInstallation 确认是否安装模块
func (md *ModuleDownloader) confirmInstallation(updateInfo *UpdateInfo, filePath string) bool {
	fmt.Println("\n" + strings.Repeat("━", 60))
	fmt.Println("📦 模块下载完成！")
	fmt.Printf("📄 模块版本: %s\n", updateInfo.Version)
	fmt.Printf("🔢 版本代码: %d\n", updateInfo.VersionCode)
	fmt.Printf("📁 文件路径: %s\n", filePath)
	if updateInfo.Changelog != "" {
		fmt.Printf("📋 更新日志: %s\n", updateInfo.Changelog)
	}
	fmt.Println(strings.Repeat("━", 60))

	fmt.Print("❓ 是否立即安装此模块？[Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

// handleGetCommand 处理get命令
func handleGetCommand(repoArg string) {
	md := NewModuleDownloader()

	// 规范化仓库名称
	repo := md.normalizeRepoName(repoArg)
	if repo == "" {
		fmt.Printf("❌ 无效的仓库格式: %s\n", repoArg)
		fmt.Println("正确格式: username/repo 或 username\\repo")
		return
	}

	fmt.Printf("🎯 目标仓库: %s\n", repo)

	// 下载update.json
	updateInfo, err := md.downloadUpdateJSON(repo)
	if err != nil {
		fmt.Printf("❌ 下载更新信息失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 获取到模块信息: %s (版本代码: %d)\n", updateInfo.Version, updateInfo.VersionCode)

	// 下载模块文件
	filePath, err := md.downloadModule(updateInfo)
	if err != nil {
		fmt.Printf("❌ 下载模块失败: %v\n", err)
		return
	}

	// 确认安装
	if md.confirmInstallation(updateInfo, filePath) {
		fmt.Println("\n🚀 开始安装模块...")
		installModule(filePath)
	} else {
		fmt.Println("⏸️  已取消安装，模块文件已保存")
		fmt.Printf("📁 文件位置: %s\n", filePath)
		fmt.Println("💡 您可以稍后使用以下命令手动安装:")
		fmt.Printf("   rmmp module install \"%s\"\n", filePath)
	}
}
