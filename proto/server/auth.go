/*
 * Copyright 2012 Nan Deng
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

package server

import (
	"crypto/rsa"
	. "github.com/uniqush/uniqush-conn/proto"
	"io"
	"net"
)

type Authenticator interface {
	Authenticate(srv, usr, token string) (bool, error)
}

var ErrAuthFail = errors.New("authentication failed")

func AuthConn(conn net.Conn, privkey *rsa.PrivateKey, auth Authenticator, timeout time.Duration) (c Conn, err error) {
	conn.SetDeadline(time.Now().Add(timeout))
	defer conn.SetDeadline(time.Time{})

	ks, err := ServerKeyExchange(privkey, conn)
	if err != nil {
		return
	}
	cmdio := ks.getServerCommandIO(conn)
	cmd, err := cmdio.ReadCommand()
	if err != nil {
		return
	}
	if cmd.Type != cmdtype_AUTH {
		return
	}
	if len(cmd.Params) != 3 {
		return
	}
	service := cmd.Params[0]
	username := cmd.Params[1]
	token := cmd.Params[2]

	// Username and service should not contain "\n"
	if strings.Contains(service, "\n") || strings.Contains(username, "\n") {
		err = ErrAuthFail
		return
	}

	ok, err := auth.Authenticate(service, username, token)
	if err != nil {
		return
	}
	if !ok {
		err = ErrAuthFail
		return
	}

	cmd.Type = cmdtype_AUTHOK
	cmd.Params = nil
	cmd.Message = nil
	err = cmdio.WriteCommand(cmd, false, true)
	if err != nil {
		return
	}
	c = newMessageChannel(cmdio, service, username, conn)
	err = nil
	return
}