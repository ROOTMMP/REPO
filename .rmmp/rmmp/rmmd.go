package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ModuleInfo 表示模块信息的结构
type ModuleInfo struct {
	ID          string `json:"id"`
	UpdateJSON  string `json:"updateJson"`
	VersionCode string `json:"versionCode"`
	Description string `json:"description"`
	Enabled     string `json:"enabled"`
	Update      string `json:"update"`
	Name        string `json:"name"`
	Web         string `json:"web"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	DirID       string `json:"dir_id"`
	Action      string `json:"action"`
	Remove      string `json:"remove"`
}

// RootEnvironment 表示Root环境类型
type RootEnvironment int

const (
	RootUnknown RootEnvironment = iota
	RootMagisk
	RootAPatch
	RootKernelSU
)

// RMMD Root模块管理器守护进程
type RMMD struct {
	rootEnv    RootEnvironment
	binaryPath string
}

// NewRMMD 创建新的RMMD实例
func NewRMMD() *RMMD {
	rmmd := &RMMD{}
	rmmd.detectRootEnvironment()
	return rmmd
}

// detectRootEnvironment 检测Root环境类型
func (r *RMMD) detectRootEnvironment() {
	// 检查文件夹是否存在来判断Root环境类型
	if r.dirExists("/data/adb/magisk") {
		r.rootEnv = RootMagisk
		r.binaryPath = "/data/adb/magisk/magisk"
		fmt.Println("🔍 检测到 Magisk 环境")
	} else if r.dirExists("/data/adb/ap") {
		r.rootEnv = RootAPatch
		r.binaryPath = "/data/adb/apd"
		fmt.Println("🔍 检测到 APatch 环境")
	} else if r.dirExists("/data/adb/ksu") {
		r.rootEnv = RootKernelSU
		r.binaryPath = "/data/adb/ksud"
		fmt.Println("🔍 检测到 KernelSU 环境")
	} else {
		r.rootEnv = RootUnknown
		fmt.Println("⚠️  未检测到支持的Root环境")
	}
}

// dirExists 检查目录是否存在
func (r *RMMD) dirExists(path string) bool {
	if runtime.GOOS != "android" && strings.HasPrefix(path, "/data/adb/") {
		// 非Android环境下跳过检测
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// fileExists 检查文件是否存在
func (r *RMMD) fileExists(path string) bool {
	if runtime.GOOS != "android" && strings.HasPrefix(path, "/data/adb/") {
		// 非Android环境下跳过检测
		return false
	}

	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ListModules 列出已安装的模块
func (r *RMMD) ListModules() ([]ModuleInfo, error) {
	switch r.rootEnv {
	case RootMagisk:
		return r.listMagiskModules()
	case RootAPatch:
		return r.listModulesViaCommand("module", "list")
	case RootKernelSU:
		return r.listModulesViaCommand("module", "list")
	default:
		return nil, fmt.Errorf("未检测到支持的Root环境")
	}
}

// listMagiskModules 列出Magisk模块（自己实现）
func (r *RMMD) listMagiskModules() ([]ModuleInfo, error) {
	modules := []ModuleInfo{}
	modulesDir := "/data/adb/modules"

	if !r.dirExists(modulesDir) {
		return modules, fmt.Errorf("模块目录不存在: %s", modulesDir)
	}

	// 遍历模块目录
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return nil, fmt.Errorf("读取模块目录失败: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		moduleID := entry.Name()
		modulePath := filepath.Join(modulesDir, moduleID)

		// 跳过一些特殊目录
		if moduleID == "lost+found" || strings.HasPrefix(moduleID, ".") {
			continue
		}

		moduleInfo, err := r.parseMagiskModule(moduleID, modulePath)
		if err != nil {
			fmt.Printf("⚠️  解析模块 %s 失败: %v\n", moduleID, err)
			continue
		}

		modules = append(modules, *moduleInfo)
	}

	return modules, nil
}

// parseMagiskModule 解析Magisk模块信息
func (r *RMMD) parseMagiskModule(moduleID, modulePath string) (*ModuleInfo, error) {
	propFile := filepath.Join(modulePath, "module.prop")

	// 检查module.prop是否存在
	if !r.fileExists(propFile) {
		return nil, fmt.Errorf("module.prop 不存在")
	}

	// 读取module.prop文件
	content, err := os.ReadFile(propFile)
	if err != nil {
		return nil, fmt.Errorf("读取module.prop失败: %v", err)
	}

	// 解析module.prop
	props := r.parseProperties(string(content))

	// 检查模块是否启用
	enabled := "true"
	disableFile := filepath.Join(modulePath, "disable")
	if r.fileExists(disableFile) {
		enabled = "false"
	}

	// 检查是否有更新
	updateJSON := props["updateJson"]
	hasUpdate := "false"
	if updateJSON != "" {
		hasUpdate = "true" // 简化处理，实际应该检查版本
	}

	module := &ModuleInfo{
		ID:          moduleID,
		UpdateJSON:  updateJSON,
		VersionCode: props["versionCode"],
		Description: props["description"],
		Enabled:     enabled,
		Update:      hasUpdate,
		Name:        props["name"],
		Web:         "false", // Magisk模块通常没有web界面
		Version:     props["version"],
		Author:      props["author"],
		DirID:       moduleID,
		Action:      "true",  // 假设都有action
		Remove:      "false", // 假设都可以删除
	}

	return module, nil
}

// parseProperties 解析properties格式的文件内容
func (r *RMMD) parseProperties(content string) map[string]string {
	props := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			props[key] = value
		}
	}

	return props
}

// listModulesViaCommand 通过命令行列出模块
func (r *RMMD) listModulesViaCommand(args ...string) ([]ModuleInfo, error) {
	if !r.fileExists(r.binaryPath) {
		return nil, fmt.Errorf("二进制文件不存在: %s", r.binaryPath)
	}

	cmd := exec.Command(r.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("执行命令失败: %v", err)
	}

	var modules []ModuleInfo
	if err := json.Unmarshal(output, &modules); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %v", err)
	}

	return modules, nil
}

// InstallModule 安装模块
func (r *RMMD) InstallModule(zipPath string) error {
	if r.rootEnv == RootUnknown {
		return fmt.Errorf("未检测到支持的Root环境")
	}

	if !r.fileExists(r.binaryPath) {
		return fmt.Errorf("二进制文件不存在: %s", r.binaryPath)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(zipPath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	fmt.Printf("🚀 使用 %s 安装模块: %s\n", r.getRootEnvName(), absPath)

	var cmd *exec.Cmd
	switch r.rootEnv {
	case RootMagisk:
		cmd = exec.Command(r.binaryPath, "--install-module", absPath)
	case RootAPatch, RootKernelSU:
		cmd = exec.Command(r.binaryPath, "module", "install", absPath)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("安装失败: %v", err)
	}

	fmt.Println("✅ 模块安装完成!")
	return nil
}

// getRootEnvName 获取Root环境名称
func (r *RMMD) getRootEnvName() string {
	switch r.rootEnv {
	case RootMagisk:
		return "Magisk"
	case RootAPatch:
		return "APatch"
	case RootKernelSU:
		return "KernelSU"
	default:
		return "Unknown"
	}
}

// PrintModuleList 打印模块列表（格式化输出）
func (r *RMMD) PrintModuleList() error {
	modules, err := r.ListModules()
	if err != nil {
		return err
	}

	if len(modules) == 0 {
		fmt.Println("📋 没有找到已安装的模块")
		return nil
	}

	fmt.Printf("📋 已安装的模块列表 (%s) - 共 %d 个:\n", r.getRootEnvName(), len(modules))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, module := range modules {
		status := "🔴 已禁用"
		if module.Enabled == "true" {
			status = "🟢 已启用"
		}

		fmt.Printf("%d. %s (%s)\n", i+1, module.Name, module.ID)
		fmt.Printf("   版本: %s (代码: %s)\n", module.Version, module.VersionCode)
		fmt.Printf("   作者: %s\n", module.Author)
		fmt.Printf("   状态: %s\n", status)
		if module.Description != "" {
			fmt.Printf("   描述: %s\n", module.Description)
		}
		if module.UpdateJSON != "" {
			updateStatus := "🔄 有更新"
			if module.Update == "false" {
				updateStatus = "✅ 最新版本"
			}
			fmt.Printf("   更新: %s\n", updateStatus)
		}
		fmt.Println("   ────────────────────────────────────────")
	}

	return nil
}
