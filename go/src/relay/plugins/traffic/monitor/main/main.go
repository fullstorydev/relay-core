package main

type monitorPlugin struct {
}

func (mp monitorPlugin) Name() string {
	return "Montor"
}

var Plugin monitorPlugin
