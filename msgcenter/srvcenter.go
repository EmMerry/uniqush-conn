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

package msgcenter

import (
	"fmt"
	"github.com/uniqush/uniqush-conn/config"
	"github.com/uniqush/uniqush-conn/proto/server"
	"github.com/uniqush/uniqush-conn/rpc"
	"io"
	"strings"
)

type serviceCenter struct {
	config     *config.ServiceConfig
	fwdChan    chan<- *rpc.ForwardRequest
	subReqChan chan *rpc.SubscribeRequest
	conns      connMap
	peer       rpc.UniqushConnPeer
}

func (self *serviceCenter) serveConn(conn server.Conn) {
	var reason error
	defer func() {
		self.conns.DelConn(conn)
		self.config.OnLogout(conn, reason)
		conn.Close()
	}()
	for {
		msg, err := conn.ReceiveMessage()
		if err != nil {
			if err != io.EOF {
				self.config.OnError(conn, err)
				reason = err
			}
		}
		if msg != nil {
			self.config.OnMessage(conn, msg)
		}
	}
}

func (self *serviceCenter) NewConn(conn server.Conn) {
	if conn == nil {
		//self.config.OnError(conn, fmt.Errorf("Nil conn")
		return
	}
	usr := conn.Username()
	if len(usr) == 0 || strings.Contains(usr, ":") || strings.Contains(usr, "\n") {
		self.config.OnError(conn, fmt.Errorf("invalid username"))
		conn.Close()
		return
	}
	conn.SetMessageCache(self.config.Cache())
	conn.SetForwardRequestChannel(self.fwdChan)
	conn.SetSubscribeRequestChan(self.subReqChan)
	err := self.conns.AddConn(conn)
	if err != nil {
		self.config.OnError(conn, err)
		conn.Close()
		return
	}

	go self.serveConn(conn)
	return
}

func (self *serviceCenter) Send(req *rpc.SendRequest) *rpc.Result {
	ret := new(rpc.Result)

	if req == nil {
		ret.Error = fmt.Errorf("invalid request")
		return ret
	}
	if req.Message == nil || req.Message.IsEmpty() {
		ret.Error = fmt.Errorf("invalid request: empty message")
		return ret
	}
	if req.Receiver == "" {
		ret.Error = fmt.Errorf("invalid request: no receiver")
		return ret
	}

	conns := self.conns.GetConn(req.Receiver)
	mid := req.Id
	msg := req.Message

	if !req.DontCache {
		mc := &rpc.MessageContainer{
			Sender:        "",
			SenderService: "",
			Message:       msg,
		}
		mid, ret.Error = self.config.CacheMessage(req.Receiver, mc, req.TTL)
		if ret.Error != nil {
			return ret
		}
	}

	if len(mid) == 0 {
		ret.Error = fmt.Errorf("undefined message Id")
		return ret
	}
	n := 0

	for _, minc := range conns {
		if conn, ok := minc.(server.Conn); ok {
			err := conn.SendMessage(msg, mid, nil)
			ret.Append(conn, err)
			if err != nil {
				self.conns.DelConn(minc)
				conn.Close()
			} else {
				n++
			}
		} else {
			self.conns.DelConn(minc)
		}
	}

	shouldPush := !req.DontPush

	// Don't push the message. We will push it on this node.
	req.DontPush = true
	// Don't cache the message. We have already cached it.
	req.DontCache = true
	req.Id = mid
	r := self.peer.Send(req)
	n += r.NrSuccess()
	ret.Join(r)

	if n == 0 && shouldPush {
		self.config.Push(req.Receiver, "", "", req.PushInfo, mid, msg.Size())
	}
	return ret
}

func (self *serviceCenter) Forward(req *rpc.ForwardRequest) *rpc.Result {
	ret := new(rpc.Result)

	if req == nil {
		ret.Error = fmt.Errorf("invalid request")
		return ret
	}
	if req.Message == nil || req.Message.IsEmpty() {
		ret.Error = fmt.Errorf("invalid request: empty message")
		return ret
	}
	if req.Receiver == "" {
		ret.Error = fmt.Errorf("invalid request: no receiver")
		return ret
	}

	conns := self.conns.GetConn(req.Receiver)
	mid := req.Id
	msg := req.Message
	mc := &req.MessageContainer

	var pushInfo map[string]string
	var shouldForward bool
	shouldPush := !req.DontPush

	if !req.DontAsk {
		// We need to ask for permission to forward this message.
		// This means the forward request is generated directly from a user,
		// not from a uniqush-conn node in a cluster.

		mc.Id = ""
		shouldForward, shouldPush, pushInfo = self.config.ShouldForward(req)

		if !shouldForward {
			return nil
		}
	}

	if !req.DontCache {
		mid, ret.Error = self.config.CacheMessage(req.Receiver, mc, req.TTL)
		if ret.Error != nil {
			return ret
		}
	}

	if len(mid) == 0 {
		ret.Error = fmt.Errorf("undefined message Id")
		return ret
	}
	n := 0

	for _, minc := range conns {
		if conn, ok := minc.(server.Conn); ok {
			err := conn.ForwardMessage(req.Sender, req.SenderService, msg, mid)
			ret.Append(conn, err)
			if err != nil {
				self.conns.DelConn(minc)
				conn.Close()
			} else {
				n++
			}
		} else {
			self.conns.DelConn(minc)
		}
	}

	// forward the message if possible,
	// Don't ask the permission to forward (we have already got the permission)
	req.DontAsk = true
	// And don't push the message. We will push it on this node.
	req.DontPush = true
	// Dont' cache it
	req.DontCache = true
	req.Id = mid
	r := self.peer.Forward(req)
	n += r.NrSuccess()
	ret.Join(r)

	if n == 0 && shouldPush {
		self.config.Push(req.Receiver, req.SenderService, req.Sender, pushInfo, mid, msg.Size())
	}
	return ret
}

func (self *serviceCenter) processSubscription() {
	for req := range self.subReqChan {
		if req == nil {
			continue
		}
		go self.config.Subscribe(req)
	}
}

func newServiceCenter(conf *config.ServiceConfig, fwdChan chan<- *rpc.ForwardRequest, peer rpc.UniqushConnPeer) *serviceCenter {
	if conf == nil || fwdChan == nil {
		return nil
	}
	ret := new(serviceCenter)
	ret.config = conf
	ret.conns = newTreeBasedConnMap(conf.MaxNrConns, conf.MaxNrUsers, conf.MaxNrConnsPerUser)
	ret.fwdChan = fwdChan
	ret.subReqChan = make(chan *rpc.SubscribeRequest)
	ret.peer = peer

	go ret.processSubscription()
	return ret
}
