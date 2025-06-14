package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	version = "0.3.5"
)

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]
	switch command {
	case "module":
		if len(os.Args) < 3 {
			fmt.Println("错误: 缺少子命令")
			showModuleHelp()
			return
		}
		handleModuleCommand(os.Args[2:])
	case "get":
		var repo string
		if len(os.Args) < 3 {
			// 默认为ROOTMMP/rmmp (自我更新)
			repo = "ROOTMMP/rmmp"
			fmt.Println("🔄 未指定仓库，默认进行自我更新...")
		} else {
			repo = os.Args[2]
		}
		handleGetCommand(repo)
	case "proxy":
		handleProxyCommand(os.Args[2:])
	case "search":
		handleSearchCommand(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("rmmp version %s\n", version)
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Printf("未知命令: %s\n", command)
		showHelp()
	}
}

// 处理模块相关命令
func handleModuleCommand(args []string) {
	if len(args) < 1 {
		showModuleHelp()
		return
	}

	subCommand := args[0]
	switch subCommand {
	case "install":
		if len(args) < 2 {
			fmt.Println("错误: 请指定要安装的zip文件")
			fmt.Println("用法: rmmp module install <module.zip>")
			return
		}
		installModule(args[1])
	case "list":
		listModules()
	default:
		fmt.Printf("未知的模块子命令: %s\n", subCommand)
		showModuleHelp()
	}
}

// 安装模块的核心逻辑
func installModule(zipFile string) {
	// 检查zip文件是否存在
	if !fileExists(zipFile) {
		fmt.Printf("错误: 文件不存在: %s\n", zipFile)
		return
	}

	// 检查文件扩展名
	if !strings.HasSuffix(strings.ToLower(zipFile), ".zip") {
		fmt.Printf("警告: 文件可能不是zip格式: %s\n", zipFile)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(zipFile)
	if err != nil {
		fmt.Printf("错误: 无法获取文件绝对路径: %v\n", err)
		return
	}

	fmt.Printf("正在安装模块: %s\n", absPath)
	fmt.Println("🔧 使用内置模块安装器...")

	// 使用内置的模块安装器
	err = installModuleWithBuiltinInstaller(absPath)
	if err != nil {
		fmt.Printf("❌ 模块安装失败: %v\n", err)
		return
	}

	fmt.Println("✅ 模块安装完成!")
}

// installModuleWithBuiltinInstaller 使用内置安装器安装模块
func installModuleWithBuiltinInstaller(zipPath string) error {
	fmt.Println("📦 正在解析模块...")

	// 使用 RMMD 内置安装器
	rmmd := NewRMMD()
	return rmmd.InstallModule(zipPath)
}

// 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// listModules 列出已安装的模块
func listModules() {
	rmmd := NewRMMD()
	err := rmmd.PrintModuleList()
	if err != nil {
		fmt.Printf("❌ 列出模块失败: %v\n", err)
	}
}

// 处理搜索命令 (待开发)
func handleSearchCommand(args []string) {
	fmt.Println("🔍 搜索功能")
	fmt.Println("此功能正在开发中，敬请期待！")
	fmt.Println("")
	fmt.Println("计划支持的功能:")
	fmt.Println("  • 搜索在线模块仓库")
	fmt.Println("  • 按名称/标签搜索模块")
	fmt.Println("  • 显示模块详细信息")
	fmt.Println("  • 直接下载安装模块")

	if len(args) > 0 {
		fmt.Printf("您搜索的关键词: %s\n", strings.Join(args, " "))
	}
}

// 显示主帮助信息
func showHelp() {
	fmt.Printf("rmmp - Root Module Manager Plus (rmm project) v%s\n", version)
	fmt.Println("一个支持多种Root模块管理器的命令行工具")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  rmmp <命令> [选项...]")
	fmt.Println("")
	fmt.Println("可用命令:")
	fmt.Println("  module    模块管理操作")
	fmt.Println("  get       下载并安装GitHub仓库的模块")
	fmt.Println("  proxy     GitHub代理管理")
	fmt.Println("  search    搜索模块 (开发中)")
	fmt.Println("  version   显示版本信息")
	fmt.Println("  help      显示帮助信息")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  rmmp module install example.zip")
	fmt.Println("  rmmp module list")
	fmt.Println("  rmmp get username/repo")
	fmt.Println("  rmmp get                    # 自我更新")
	fmt.Println("  rmmp proxy list")
	fmt.Println("  rmmp search keyword")
	fmt.Println("  rmmp version")
	fmt.Println("")
	fmt.Println("获取特定命令的帮助:")
	fmt.Println("  rmmp module help")
}

// 显示模块命令帮助
func showModuleHelp() {
	fmt.Println("rmmp module - 模块管理操作")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  rmmp module <子命令> [选项...]")
	fmt.Println("")
	fmt.Println("可用子命令:")
	fmt.Println("  install <zip文件>   安装指定的模块zip文件")
	fmt.Println("  list                列出已安装的模块")
	fmt.Println("")
	fmt.Println("特性:")
	fmt.Println("  • 内置模块安装器，无需外部依赖")
	fmt.Println("  • 支持多种Root环境 (KernelSU, APatch, Magisk)")
	fmt.Println("  • 自动模块验证和冲突检测")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  rmmp module install /sdcard/module.zip")
	fmt.Println("  rmmp module install ./local-module.zip")
	fmt.Println("  rmmp module list")
}

// 处理代理相关命令
func handleProxyCommand(args []string) {
	gpm := NewGitHubProxyManager()

	if len(args) < 1 {
		showProxyHelp()
		return
	}

	subCommand := args[0]

	switch subCommand {
	case "list", "ls":
		err := gpm.ListProxies()
		if err != nil {
			fmt.Printf("❌ 获取代理列表失败: %v\n", err)
		}
	case "best":
		bestProxy, err := gpm.GetBestProxy()
		if err != nil {
			fmt.Printf("❌ 获取最佳代理失败: %v\n", err)
			return
		}
		fmt.Printf("⭐ 最佳GitHub代理: %s\n", bestProxy.URL)
		fmt.Printf("   服务商: %s\n", bestProxy.Server)
		fmt.Printf("   IP地址: %s\n", bestProxy.IP)
		fmt.Printf("   延迟: %dms\n", bestProxy.Latency)
		fmt.Printf("   速度: %.2fMB/s\n", bestProxy.Speed)
	case "update":
		gpm.ClearCache()
		proxies, err := gpm.GetProxies()
		if err != nil {
			fmt.Printf("❌ 更新代理数据失败: %v\n", err)
			return
		}
		fmt.Printf("✅ 代理数据已更新，共获取 %d 个代理\n", len(proxies))
	case "clear":
		err := gpm.ClearCache()
		if err != nil {
			fmt.Printf("❌ 清除缓存失败: %v\n", err)
		}
	case "help", "-h", "--help":
		showProxyHelp()
	default:
		fmt.Printf("未知的代理子命令: %s\n", subCommand)
		showProxyHelp()
	}
}

// 显示代理命令帮助
func showProxyHelp() {
	fmt.Println("rmmp proxy - GitHub代理管理")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  rmmp proxy <子命令> [选项...]")
	fmt.Println("")
	fmt.Println("可用子命令:")
	fmt.Println("  list, ls      列出所有可用的GitHub代理")
	fmt.Println("  best          显示推荐的最佳代理")
	fmt.Println("  update        强制更新代理数据")
	fmt.Println("  clear         清除缓存文件")
	fmt.Println("  help          显示帮助信息")
	fmt.Println("")
	fmt.Println("特性:")
	fmt.Println("  • 自动缓存代理数据（10小时有效期）")
	fmt.Println("  • 智能推荐最佳代理（综合延迟和速度）")
	fmt.Println("  • 支持强制更新和缓存管理")
	fmt.Println("  • 跨平台支持，自动选择合适的缓存路径")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  rmmp proxy list          # 列出所有代理")
	fmt.Println("  rmmp proxy best          # 显示最佳代理")
	fmt.Println("  rmmp proxy update        # 强制更新数据")
	fmt.Println("  rmmp proxy clear         # 清除缓存")
	fmt.Println("") // 显示当前平台的缓存路径
	gpm := NewGitHubProxyManager()
	fmt.Printf("缓存文件位置: %s\n", gpm.GetCacheFilePath())
}
