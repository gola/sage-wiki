package parsers

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	if !r.Supports("go") {
		t.Error("should support .go")
	}
	if !r.Supports("ts") {
		t.Error("should support .ts")
	}
	if !r.Supports("json") {
		t.Error("should support .json")
	}
	if r.Supports("md") {
		t.Error("should not support .md")
	}

	result, err := r.Parse("unknown.md", []byte("# hello"))
	if err != nil || result != nil {
		t.Error("unsupported extension should return nil, nil")
	}
}

func TestGoParser(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os"
)

const MaxRetries = 5

type Config struct {
	Name    string
	Timeout int
}

type Reader interface {
	Read(p []byte) (int, error)
}

func main() {
	fmt.Println("hello")
}

func (c *Config) Validate() error {
	return nil
}

func helper() {}
`
	p := &GoParser{}
	result, err := p.Parse("main.go", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if result.Language != "go" {
		t.Errorf("language = %s, want go", result.Language)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}
	if len(result.Types) < 2 {
		t.Errorf("types = %d, want >= 2 (Config + Reader)", len(result.Types))
	}

	// Check Config struct has fields
	var configType *TypeDecl
	for i := range result.Types {
		if result.Types[i].Name == "Config" {
			configType = &result.Types[i]
			break
		}
	}
	if configType == nil {
		t.Fatal("Config type not found")
	}
	if configType.Kind != "struct" {
		t.Errorf("Config kind = %s, want struct", configType.Kind)
	}
	if len(configType.Fields) != 2 {
		t.Errorf("Config fields = %d, want 2", len(configType.Fields))
	}

	// Check functions
	if len(result.Functions) < 3 {
		t.Errorf("functions = %d, want >= 3", len(result.Functions))
	}

	// Check Validate has receiver
	var validateFn *FuncDecl
	for i := range result.Functions {
		if result.Functions[i].Name == "Validate" {
			validateFn = &result.Functions[i]
			break
		}
	}
	if validateFn == nil {
		t.Fatal("Validate function not found")
	}
	if validateFn.Receiver != "*Config" {
		t.Errorf("Validate receiver = %s, want *Config", validateFn.Receiver)
	}

	// Check exports (Config, Reader, MaxRetries, Validate — not main, helper)
	exportNames := make(map[string]bool)
	for _, e := range result.Exports {
		exportNames[e.Name] = true
	}
	if !exportNames["Config"] {
		t.Error("Config should be exported")
	}
	if !exportNames["Reader"] {
		t.Error("Reader should be exported")
	}
	if !exportNames["MaxRetries"] {
		t.Error("MaxRetries should be exported")
	}
	if exportNames["helper"] {
		t.Error("helper should not be exported")
	}
}

func TestTypeScriptParser(t *testing.T) {
	src := `import { Component } from 'react';
import * as fs from 'fs';

export interface Config {
  name: string;
}

export class App extends Component {
  render() { return null; }
}

export function main(): void {
  console.log("hello");
}

export const VERSION = "1.0";

type Internal = string;
`
	p := &TypeScriptParser{}
	result, err := p.Parse("app.ts", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Language != "typescript" {
		t.Errorf("language = %s, want typescript", result.Language)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}
	if len(result.Types) < 2 {
		t.Errorf("types = %d, want >= 2", len(result.Types))
	}

	exportNames := make(map[string]bool)
	for _, e := range result.Exports {
		exportNames[e.Name] = true
	}
	if !exportNames["main"] {
		t.Error("main should be exported")
	}
	if !exportNames["App"] {
		t.Error("App should be exported")
	}
	if !exportNames["VERSION"] {
		t.Error("VERSION should be exported")
	}
}

func TestPythonParser(t *testing.T) {
	src := `import os
from pathlib import Path

class Config:
    pass

def process(data):
    pass

async def fetch(url):
    pass

def _private():
    pass

@dataclass
class Item:
    name: str
`
	p := &PythonParser{}
	result, err := p.Parse("main.py", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}
	if len(result.Types) < 2 {
		t.Errorf("types = %d, want >= 2 (Config + Item)", len(result.Types))
	}

	// _private should not be exported
	for _, e := range result.Exports {
		if e.Name == "_private" {
			t.Error("_private should not be exported")
		}
	}
}

func TestRustParser(t *testing.T) {
	src := `use std::io::Read;
use crate::config::Config;

pub struct Server {
    port: u16,
}

pub enum Status {
    Running,
    Stopped,
}

pub trait Handler {
    fn handle(&self);
}

pub fn start(port: u16) {
}

fn internal() {
}
`
	p := &RustParser{}
	result, err := p.Parse("main.rs", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}

	exportNames := make(map[string]bool)
	for _, e := range result.Exports {
		exportNames[e.Name] = true
	}
	if !exportNames["Server"] {
		t.Error("Server should be exported")
	}
	if !exportNames["start"] {
		t.Error("start should be exported")
	}
	if exportNames["internal"] {
		t.Error("internal should not be exported")
	}
}

func TestJavaParser(t *testing.T) {
	src := `import java.util.List;
import java.io.IOException;

public class UserService {
    public List<User> findAll() {
        return null;
    }

    private void internal() {
    }
}
`
	p := &JavaParser{}
	result, err := p.Parse("UserService.java", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}
	if len(result.Types) != 1 {
		t.Errorf("types = %d, want 1 (UserService)", len(result.Types))
	}
}

func TestCParser(t *testing.T) {
	src := `#include <stdio.h>
#include "config.h"

struct Config {
    int port;
    char* name;
};

int main(int argc, char** argv) {
    return 0;
}

static void helper() {
}
`
	p := &CParser{}
	result, err := p.Parse("main.c", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Language != "c" {
		t.Errorf("language = %s, want c", result.Language)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}

	// Should detect as C++
	result2, _ := p.Parse("main.cpp", []byte(src))
	if result2.Language != "cpp" {
		t.Errorf("cpp language = %s, want cpp", result2.Language)
	}
}

func TestRubyParser(t *testing.T) {
	src := `require 'json'
require 'net/http'

class Server
  attr_accessor :port, :host

  def initialize(port)
    @port = port
  end

  def start
  end

  def self.default
  end
end

module Config
end
`
	p := &RubyParser{}
	result, err := p.Parse("server.rb", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(result.Imports))
	}
	if len(result.Types) != 1 {
		t.Errorf("types = %d, want 1 (Server)", len(result.Types))
	}
}

func TestJSONParser(t *testing.T) {
	src := `{
  "name": "sage-wiki",
  "version": "1.0",
  "dependencies": {
    "react": "^18.0",
    "next": "^14.0"
  }
}`
	p := &JSONParser{}
	result, err := p.Parse("package.json", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	keyNames := make(map[string]bool)
	for _, e := range result.Exports {
		keyNames[e.Name] = true
	}
	if !keyNames["name"] {
		t.Error("should find 'name' key")
	}
	if !keyNames["dependencies.react"] {
		t.Error("should find nested 'dependencies.react' key")
	}
}

func TestYAMLParser(t *testing.T) {
	src := `project: sage-wiki
compiler:
  max_parallel: 20
  mode: auto
`
	p := &YAMLParser{}
	result, err := p.Parse("config.yaml", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	keyNames := make(map[string]bool)
	for _, e := range result.Exports {
		keyNames[e.Name] = true
	}
	if !keyNames["project"] {
		t.Error("should find 'project' key")
	}
	if !keyNames["compiler.max_parallel"] {
		t.Error("should find nested 'compiler.max_parallel' key")
	}
}

func TestTOMLParser(t *testing.T) {
	src := `[package]
name = "sage-wiki"
version = "0.1.0"

[dependencies]
serde = "1.0"
`
	p := &TOMLParser{}
	result, err := p.Parse("Cargo.toml", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	keyNames := make(map[string]bool)
	for _, e := range result.Exports {
		keyNames[e.Name] = true
	}
	if !keyNames["package"] {
		t.Error("should find 'package' section")
	}
	if !keyNames["package.name"] {
		t.Error("should find 'package.name' key")
	}
	if !keyNames["dependencies.serde"] {
		t.Error("should find 'dependencies.serde' key")
	}
}
