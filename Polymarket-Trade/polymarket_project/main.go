package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"Polymarket/src/pkg/papermcp"
	"Polymarket/src/service/taker"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 {
		return fmt.Errorf("请运行 go run .，在 TUI 里填写参数")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	m := &model{ctx: ctx, form: newSetup(), width: 104, height: 24}
	_, err := tea.NewProgram(m, tea.WithContext(ctx)).Run()
	if m.session != nil {
		m.session.close()
	}
	return err
}

type session struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	views    chan taker.View
	controls chan taker.Command
}

func startSession(parent context.Context, params sessionParams, l uiLanguage) (*session, error) {
	path, err := filepath.Abs("../python_adapter/bin/pm-trader-mcp")
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("%s", l.text("找不到 Python 适配器，请从 polymarket_project 目录启动", "Python adapter missing. Run the program from the polymarket_project directory."))
	}
	if _, err := pythonExecutable(l); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parent)
	s := &session{ctx: ctx, cancel: cancel, done: make(chan struct{}, 2),
		views: make(chan taker.View, 1), controls: make(chan taker.Command, 4)}
	provider := papermcp.New(path, params.args...)
	go func() {
		defer func() { s.done <- struct{}{} }()
		provider.Run(ctx)
	}()
	go func() {
		defer func() { s.done <- struct{}{} }()
		taker.Run(ctx, provider, params.config, s.controls, s.views)
	}()
	return s, nil
}

func pythonExecutable(l uiLanguage) (string, error) {
	python := os.Getenv("PAPER_PYTHON")
	if python == "" {
		python = "../python_adapter/.venv/bin/python"
	}
	python, err := filepath.Abs(python)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(python); err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return "", fmt.Errorf("%s", l.text("找不到 Python 虚拟环境，安装步骤见 README.md", "Python environment missing. See README.md for setup."))
	}
	return python, nil
}

func (s *session) close() {
	s.cancel()
	<-s.done
	<-s.done
}
