// -*- Mode: Go; indent-tabs-mode: t -*-

package main_test

import (
	"errors"

	"github.com/snapcore/snapd/asserts"
)

type namedKeypairManager struct {
	asserts.KeypairManager
	keysByName map[string]asserts.PrivateKey
}

func newNamedKeypairManager() *namedKeypairManager {
	return &namedKeypairManager{
		KeypairManager: asserts.NewMemoryKeypairManager(),
		keysByName:     make(map[string]asserts.PrivateKey),
	}
}

func (m *namedKeypairManager) AddKey(name string, priv asserts.PrivateKey) error {
	if err := m.Put(priv); err != nil {
		return err
	}
	m.keysByName[name] = priv
	return nil
}

func (m *namedKeypairManager) GetByName(keyName string) (asserts.PrivateKey, error) {
	priv, ok := m.keysByName[keyName]
	if !ok {
		return nil, errors.New("cannot find key pair in GPG keyring")
	}
	return priv, nil
}

func (m *namedKeypairManager) Export(keyName string) ([]byte, error) {
	_, ok := m.keysByName[keyName]
	if !ok {
		return nil, errors.New("cannot find key pair in GPG keyring")
	}
	return nil, errors.New("cannot export keypair in memory key manager")
}

func (m *namedKeypairManager) List() ([]asserts.ExternalKeyInfo, error) {
	return nil, nil
}

func (m *namedKeypairManager) DeleteByName(keyName string) error {
	priv, err := m.GetByName(keyName)
	if err != nil {
		return err
	}
	if err := m.Delete(priv.PublicKey().ID()); err != nil {
		return err
	}
	delete(m.keysByName, keyName)
	return nil
}
