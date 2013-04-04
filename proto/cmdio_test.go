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
	"testing"
	"fmt"
)

func testSendingCommands(t *testing.T, from, to *commandIO, cmds ...*command) {
	errCh := make(chan error)
	go func() {
		for i, cmd := range cmds {
			recved, err := to.ReadCommand()
			if err != nil {
				errCh <- err
				return
			}
			if !cmd.eq(recved) {
				errCh <- fmt.Errorf("%vth command does not equal", i)
			}
		}
		close(errCh)
	}()

	for _, cmd := range cmds {
		err := from.WriteCommand(cmd)
		if err != nil {
			t.Errorf("Error on write: %v", err)
		}
	}

	for err := range errCh {
		if err != nil {
			t.Errorf("Error on write: %v", err)
		}
	}
}

func testExchangingCommands(t *testing.T, compress, encrypt bool, cmds ...*command) {
	sks, cks, s2c, c2s := exchangeKeysOrReport(t)
	if sks == nil || cks == nil || s2c == nil || c2s == nil {
		return
	}

	scmdio := sks.getServerCommandIO(s2c)
	if !compress {
		scmdio.ReadCompressOff()
		scmdio.WriteCompressOff()
	}
	if !encrypt {
		scmdio.ReadEncryptOff()
		scmdio.WriteEncryptOff()
	}

	ccmdio := cks.getServerCommandIO(c2s)
	if !compress {
		ccmdio.ReadCompressOff()
		ccmdio.WriteCompressOff()
	}
	if !encrypt {
		ccmdio.ReadEncryptOff()
		ccmdio.WriteEncryptOff()
	}
	testSendingCommands(t, scmdio, ccmdio, cmds...)
	testSendingCommands(t, ccmdio, scmdio, cmds...)
}

func TestExchangingFullCommandNoCompressNoEncrypt(t *testing.T) {
	cmd := new(command)
	cmd.Body = []byte{1,2,3}
	cmd.Type = 1
	cmd.Params = make([][]byte, 2)
	cmd.Params[0] = []byte{1,2,3}
	cmd.Params[1] = []byte{2,2,3}
	cmd.Header = make(map[string]string, 2)
	cmd.Header["a"] = "hello"
	cmd.Header["b"] = "hell"
	testExchangingCommands(t, false, false, cmd)
}

func TestExchangingFullCommandNoCompress(t *testing.T) {
	cmd := new(command)
	cmd.Body = []byte{1,2,3}
	cmd.Type = 1
	cmd.Params = make([][]byte, 2)
	cmd.Params[0] = []byte{1,2,3}
	cmd.Params[1] = []byte{2,2,3}
	cmd.Header = make(map[string]string, 2)
	cmd.Header["a"] = "hello"
	cmd.Header["b"] = "hell"
	testExchangingCommands(t, true, false, cmd)
}

func TestExchangingFullCommand(t *testing.T) {
	cmd := new(command)
	cmd.Body = []byte{1,2,3}
	cmd.Type = 1
	cmd.Params = make([][]byte, 2)
	cmd.Params[0] = []byte{1,2,3}
	cmd.Params[1] = []byte{2,2,3}
	cmd.Header = make(map[string]string, 2)
	cmd.Header["a"] = "hello"
	cmd.Header["b"] = "hell"
	testExchangingCommands(t, true, true, cmd)
}
