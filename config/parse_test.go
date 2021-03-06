/*
 * Copyright 2013 Nan Deng
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package config

import (
	"bytes"
	"os"
	"testing"
)

var configFileContent string = `
# This is a comment
http-addr: 127.0.0.1:8088
handshake-timeout: 10s
auth:
  default: disallow
  url: http://localhost:8080/auth
  timeout: 3s
err: 
  url: http://localhost:8080/err
  timeout: 3s
default:
  msg:
  # A web hook takes either a url or a list of url
    url:
      - http://localhost:8080/msg
      - http://localhost:8081/msg
    timeout: 3s
  err: 
    url: http://localhost:8080/err
    timeout: 3s
  login: 
    url: http://localhost:8080/login
    timeout: 3s
  logout: 
    url: http://localhost:8080/logout
    timeout: 3s
  fwd: 
    default: allow
    url: http://localhost:8080/fwd
    timeout: 3s
    max-ttl: 36h
  subscribe:
    default: allow
    url: http://localhost:8080/subscribe
    timeout: 3s
  unsubscribe:
    default: allow
    url: http://localhost:8080/unsubscribe
    timeout: 3s
  uniqush-push:
    addr: localhost:9898
    timeout: 3s
  max-conns: 2048
  max-online-users: 2048
  max-conns-per-user: 10
  db:
    engine: redis
    host: 127.0.0.1
    port: 6379
    database: 1
    `

func writeConfigFile(filename string) {
	file, _ := os.Create(filename)
	file.WriteString(configFileContent)
	file.Close()
}

func deleteConfigFile(filename string) {
	os.Remove(filename)
}

func TestParseFile(t *testing.T) {
	filename := "config.yaml"
	writeConfigFile(filename)
	defer deleteConfigFile(filename)
	_, err := ParseFile(filename)
	if err != nil {
		t.Errorf("Error: %v\n", err)
	}
}

func TestParseReader(t *testing.T) {
	reader := bytes.NewBufferString(configFileContent)
	_, err := Parse(reader)
	if err != nil {
		t.Errorf("Error: %v\n", err)
	}
}
