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

package rpc

import "net"

type Result struct {
	Error   string        `json:"error,omitempty"`
	Results []*ConnResult `json:"results,omitempty"`
}

func (self *Result) SetError(err error) {
	if self == nil || err == nil {
		return
	}
	self.Error = err.Error()
}

func (self *Result) NrResults() int {
	if self == nil {
		return 0
	}
	return len(self.Results)
}

func (self *Result) NrSuccess() int {
	if self == nil {
		return 0
	}
	ret := 0
	for _, r := range self.Results {
		if r.Error == "" {
			ret++
		}
	}
	return ret
}

func (self *Result) NrSuccessForUser(service, user string) int {
	if self == nil {
		return 0
	}
	ret := 0
	for _, r := range self.Results {
		if r.Service == service && r.Username == user && r.Error == "" {
			ret += 1
		}
	}
	return ret
}

func (self *Result) Join(r *Result) {
	if self == nil {
		return
	}
	if r == nil {
		return
	}
	if self.Error != "" {
		return
	}
	if r.Error != "" {
		self.Error = r.Error
		return
	}
	self.Results = append(self.Results, r.Results...)
}

type connDescriptor interface {
	RemoteAddr() net.Addr
	Service() string
	Username() string
	UniqId() string
	Visible() bool
}

func (self *Result) Append(c connDescriptor, err error) {
	if self == nil {
		return
	}
	if self.Results == nil {
		self.Results = make([]*ConnResult, 0, 10)
	}
	r := new(ConnResult)
	r.ConnId = c.UniqId()
	if err != nil {
		r.Error = err.Error()
	}
	r.Visible = c.Visible()
	r.Username = c.Username()
	r.Service = c.Service()
	r.Address = c.RemoteAddr().String()
	self.Results = append(self.Results, r)
}

type ConnResult struct {
	Address  string `json:"address"`
	ConnId   string `json:"conn-id"`
	Error    string `json:"error,omitempty"`
	Visible  bool   `json:"visible"`
	Username string `josn:"username"`
	Service  string `json:"service"`
}
