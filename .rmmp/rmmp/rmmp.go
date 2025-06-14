package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	version    = "0.3.5"
	ksudPath   = "/data/adb/ksud"
	apdPath    = "/data/adb/apd"
	magiskPath = "/data/adb/magisk"
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

	// 按优先级检查并执行安装
	if fileExists(ksudPath) {
		fmt.Println("检测到 KernelSU，使用 ksud 安装模块...")
		executeCommand(ksudPath, "module", "install", absPath)
	} else if fileExists(apdPath) {
		fmt.Println("检测到 APatch，使用 apd 安装模块...")
		executeCommand(apdPath, "module", "install", absPath)
	} else if fileExists(magiskPath) {
		fmt.Println("检测到 Magisk，使用 magisk 安装模块...")
		executeCommand(magiskPath, "--install-module", absPath)
	} else {
		printWarning()
	}
}

// 执行系统命令
func executeCommand(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("执行命令: %s %s\n", command, strings.Join(args, " "))

	err := cmd.Run()
	if err != nil {
		fmt.Printf("命令执行失败: %v\n", err)
		return
	}

	fmt.Println("模块安装完成!")
}

// 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// 打印警告信息
func printWarning() {
	fmt.Println("⚠️  警告: 未检测到支持的模块管理器!")
	fmt.Println("")
	fmt.Println("rmmp 支持以下模块管理器:")
	fmt.Println("  • KernelSU (ksud)")
	fmt.Println("  • APatch (apd)")
	fmt.Println("  • Magisk")
	fmt.Println("")
	fmt.Println("请确保您的设备已安装其中一种模块管理器，")
	fmt.Println("并且相关二进制文件位于以下路径之一:")
	fmt.Printf("  • %s\n", ksudPath)
	fmt.Printf("  • %s\n", apdPath)
	fmt.Printf("  • %s\n", magiskPath)
	fmt.Println("")
	fmt.Println("如果您确定已安装模块管理器但仍看到此警告，")
	fmt.Println("请检查路径是否正确或联系开发者更新支持。")
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
	fmt.Println("  search    搜索模块 (开发中)")
	fmt.Println("  version   显示版本信息")
	fmt.Println("  help      显示帮助信息")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  rmmp module install example.zip")
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
	fmt.Println("")
	fmt.Println("支持的模块管理器 (按优先级):")
	fmt.Println("  1. KernelSU (ksud)")
	fmt.Println("  2. APatch (apd)")
	fmt.Println("  3. Magisk")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  rmmp module install /sdcard/module.zip")
	fmt.Println("  rmmp module install ./local-module.zip")
}
