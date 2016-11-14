/*
Copyright 2012 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package idempotency provides a duplicate function call suppression
// mechanism. It is a lightly modified version of groupcache's
// singleflight package that does not forget keys until explicitly
// told to.
package idempotency

import "sync"

// call is an in-flight or completed Once call
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Once executes and returns the results of the given function, making
// sure that only one execution for a given key happens until the
// key is explicitly forgotten. If a duplicate comes in, the duplicate
// caller waits for the original to complete and receives the same results.
func (g *Group) Once(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	if c.err != nil {
		g.mu.Lock()
		delete(g.m, key)
		g.mu.Unlock()
	}
	c.wg.Done()

	return c.val, c.err
}

// Forget forgets a key, allowing the next call for the key to execute
// the function.
func (g *Group) Forget(key string) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}
