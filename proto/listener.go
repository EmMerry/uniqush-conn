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

package proto

import (
	"net"
	"io"
	"errors"
	"crypto/sha256"
	"crypto/rsa"
	"crypto/rand"
)

var ErrBadKeyExchangePacket = errors.New("Bad Key-exchange Packet")

type authResult struct {
	sessionKey []byte
	mackey []byte
	err error
	c net.Conn
}

type Authenticator interface {
	Authenticate(usr, token string) (bool, error)
}

type serverListener struct {
	listener net.Listener
	auth Authenticator
	privKey *rsa.PrivateKey
}

func Listen(listener net.Listener, auth Authenticator, privKey *rsa.PrivateKey) (l net.Listener, err error) {
	ret := new(serverListener)
	ret.listener = listener
	ret.auth = auth
	ret.privKey = privKey
	l = ret
	return
}

func (self *serverListener) Addr() net.Addr {
	return self.listener.Addr()
}

func (self *serverListener) Close() error {
	err := self.listener.Close()
	return err
}

func (self *serverListener) auth(conn net.Conn) *authResult {
	// Since we are using RSA-OAEP encryption,
	// there are 256 bytes for each block
	keyExPktLen := 256
	keyExPkt := make([]byte, keyExPktLen)
	ret := new(authResult)

	// Let's first read the keys.
	n, err := io.ReadFull(c, keyExPkt)

	if err != nil {
		ret.err = err
		return ret
	}
	if n != len(keyExPkt) {
		ret.err = ErrBadKeyExchangePacket
		return ret
	}

	// Now, let's decrypt it.
	// This data is not compressed
	// because they are basically random data
	// and should be hardly compressed.
	sha := sha256.New()
	keyData, err := rsa.DecryptOAEP(sha, rand.Reader, self.privKey, keyExPkt, nil)

	if err != nil {
		ret.err = err
		return ret
	}
	if len(keyData) < sessionKeyLen + macKeyLen {
		ret.err = ErrBadKeyExchangePacket
	}

	// The client send the first packet
	// with the following fields (in sequence):
	//
	// - session key.
	// - mac key.
	// - random data used to authenticate the server's identity.
	//

	randomData := keyData[sessionKeyLen + macKeyLen:]

	// We send back the random data to prove the identity
	err = writen(conn, randomData)
	if err != nil {
		ret.err = err
		return ret
	}

	// Now, it's time to copy the keys
	ret.sessionKey = make([]byte, sessionKeyLen)
	ret.macKey = make([]byte, macKeyLen)
	copy(ret.sessionKey, keyData)
	copy(ret.macKey, keyData[sessionKeyLen:]

	// TODO username/password auth
	ret.conn = conn
	return ret
}

func (self *serverListener) Accept() (conn net.Conn, err error) {
	c, err := self.listener.Accept()
	if err != nil {
		return
	}

	res := self.auth(c)

	if res.err != nil {
		err = res.err
		return
	}
	conn = res.conn
	return
}
