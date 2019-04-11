// Package database provides communication and storage for scp and web packages
package database

// sinks are peers providing files
var sinks map[string]Sink

// sources are peers receiving files
var sources map[string]Source
