package main

import (
	"context"
	"exceltranslator/config"
	"exceltranslator/core"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 处理命令行参数
	if len(os.Args) != 3 {
		fmt.Println("使用方法: ./exceltranslator input.xlsx output.xlsx")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	// 验证输入文件是否存在
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		log.Fatalf("输入文件不存在: %s", inputFile)
	}

	// 验证输入文件扩展名
	if !strings.HasSuffix(strings.ToLower(inputFile), ".xlsx") {
		log.Fatalf("输入文件必须是 .xlsx 格式: %s", inputFile)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("创建输出目录时出错: %v", err)
	}

	// 从配置文件加载配置
	_, err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置文件失败: %v", err)
	}

	// 处理单个 Excel 文件
	ctx := context.Background()
	processErr := core.ProcessExcelFile(ctx, inputFile, outputFile, func(original, translated string) {
		log.Printf("翻译: %s -> %s", original, translated)
	})
	if processErr != nil {
		log.Fatalf("处理文件时出错: %v", processErr)
	}
}
