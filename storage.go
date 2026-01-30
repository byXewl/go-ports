package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Rule 端口转发规则
type Rule struct {
	ID         string `json:"id"`
	Seq        int    `json:"seq"` // 序号，从1叠加
	ListenAddr string `json:"listenAddr"`
	ListenPort string `json:"listenPort"`
	TargetAddr string `json:"targetAddr"`
	TargetPort string `json:"targetPort"`
}

// Template 规则模板
type Template struct {
	Name      string   `json:"name"`
	Rules     []string `json:"rules"` // 存储规则ID列表
	CreatedAt string   `json:"createdAt"`
}

// AppData 应用程序数据
type AppData struct {
	Rules     []Rule     `json:"rules"`
	Templates []Template `json:"templates"`
}

// Storage 存储管理
type Storage struct {
	dataFile string
}

// NewStorage 创建新的存储管理
func NewStorage() *Storage {
	dbDir := filepath.Join(".", "db")
	return &Storage{
		dataFile: filepath.Join(dbDir, "data.json"),
	}
}

// loadAppData 加载应用程序数据
func (s *Storage) loadAppData() (AppData, error) {
	// 检查文件是否存在
	if _, err := os.Stat(s.dataFile); os.IsNotExist(err) {
		return AppData{
			Rules:     []Rule{},
			Templates: []Template{},
		}, nil
	}

	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		return AppData{}, fmt.Errorf("failed to read data file: %w", err)
	}

	var appData AppData
	if err := json.Unmarshal(data, &appData); err != nil {
		return AppData{}, fmt.Errorf("failed to unmarshal app data: %w", err)
	}

	return appData, nil
}

// saveAppData 保存应用程序数据
func (s *Storage) saveAppData(appData AppData) error {
	data, err := json.MarshalIndent(appData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal app data: %w", err)
	}

	if err := os.WriteFile(s.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	log.Printf("Saved app data: %d rules, %d templates", len(appData.Rules), len(appData.Templates))
	return nil
}

// SaveRules 保存规则
func (s *Storage) SaveRules(rules []Rule) error {
	appData, err := s.loadAppData()
	if err != nil {
		return err
	}

	appData.Rules = rules
	return s.saveAppData(appData)
}

// LoadRules 加载规则
func (s *Storage) LoadRules() ([]Rule, error) {
	appData, err := s.loadAppData()
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded %d rules", len(appData.Rules))
	return appData.Rules, nil
}

// SaveTemplates 保存模板
func (s *Storage) SaveTemplates(templates []Template) error {
	appData, err := s.loadAppData()
	if err != nil {
		return err
	}

	appData.Templates = templates
	return s.saveAppData(appData)
}

// LoadTemplates 加载模板
func (s *Storage) LoadTemplates() ([]Template, error) {
	appData, err := s.loadAppData()
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded %d templates", len(appData.Templates))
	return appData.Templates, nil
}
