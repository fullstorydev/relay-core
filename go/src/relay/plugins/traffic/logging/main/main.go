package main

type loggingPlugin struct {
}

func (lp loggingPlugin) Name() string {
	return "Logging"
}

var Plugin loggingPlugin
