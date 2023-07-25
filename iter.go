package fp

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// Next iterator type with some processing functions implemented for it.
type Next[T any] func() (T, bool)

func (next Next[T]) String() string {
	var s = strings.Split(runtime.FuncForPC(reflect.ValueOf(next).Pointer()).Name(), "/")
	return fmt.Sprintf("<%v %p>", s[len(s)-1], next)
}

// Filter the iterator, keeping only the values that evaluate to 'true'.
// this function is lazy.
func Filter[T any](next Next[T], fn func(T) bool) Next[T] {
	return func() (T, bool) {
		for {
			t, ok := next()
			if Not(ok) {
				return Zero[T](), false
			}

			if fn(t) {
				return t, true
			}
		}
	}
}

// Reduce from left to right, 1, 2, 3, 4 will be computed as (((1+2)+3)+4),
// resulting in a single value. the function is not lazy
func Reduce[T any](next Next[T], fn func(T, T) T) T {
	var poly T
	for {
		t, ok := next()
		if Not(ok) {
			return poly
		}

		poly = fn(poly, t)
	}
}

// Map the values inside the iterator to new values
func Map[R, T any](next Next[T], fn func(T) R) Next[R] {
	return func() (R, bool) {
		t, ok := next()
		if Not(ok) {
			return Zero[R](), false
		}

		return fn(t), true
	}
}

// Zip the values at the beginning of multiple iterators.
func Zip[T any](nexts ...Next[T]) Next[[]T] {
	return func() ([]T, bool) {
		var slice = make([]T, 0)
		for i := range nexts {
			t, ok := nexts[i]()
			if Not(ok) {
				return Zero[[]T](), false
			}
			slice = append(slice, t)
		}

		return slice, true
	}
}

// Lock the iterator to make it concurrent-safe.
func Lock[T any](next Next[T]) Next[T] {
	var lock sync.Mutex
	return func() (T, bool) {
		lock.Lock()
		defer lock.Unlock()

		return next()
	}
}

// Take iterate the iterator only for a specific number of times
func Take[T any](stop int, next Next[T]) Next[T] {
	var n int
	return func() (T, bool) {
		n++
		if n > stop {
			return Zero[T](), false
		}

		return next()
	}
}

// ForEach loop through an iterator to retrieve values, and you can stop the loop prematurely by
// returning false from the 'fn'.
func ForEach[T any](next Next[T], fn func(T) bool) {
	for n, ok := next(); ok; n, ok = next() {
		if Not(fn(n)) {
			return
		}
	}
}

// Stop add a stopping condition to the iterator, often used to stop infinite iterators.
func Stop[T any](next Next[T], fn func(T) bool) Next[T] {
	return func() (T, bool) {
		t, _ := next()
		if fn(t) {
			return Zero[T](), false
		}

		return t, true
	}
}

// From make the iterator compatible with the 'range'.
//
//	for v := range From(Iter){
//	  //do something
//	}
func From[T any](next Next[T]) <-chan T {
	var y = make(chan T)
	go func() {
		for n, ok := next(); ok; n, ok = next() {
			y <- n
		}

		close(y)
	}()

	return y
}

// Yield iteration in Go
func Yield[T any](next Next[T]) (yield func() T) {
	var y = From(next)
	return func() T {
		return <-y
	}
}

// Range generate a range of integers, similar to Python built-in 'range' function.
func Range[N Integer](r ...N) Next[N] {
	switch len(r) {
	case 0:
		panic(ErrLeastOne)
	case 1:
		return rangeN(Zero[N]()-1, r[0], 1)
	case 2:
		return rangeN(r[0]-1, r[1], 1)
	default:
		return rangeN(r[0]-r[2], r[1], r[2])
	}
}

func rangeN[N Integer](start, stop, step N) Next[N] {
	return func() (N, bool) {
		start += step
		if start >= stop {
			return Zero[N](), false
		}

		return start, true
	}
}

// Slice generate a slice from an iterator.
func Slice[S ~[]T, T any](i Next[T]) S {
	var slice = make(S, 0)
	ForEach(i, func(t T) bool {
		slice = append(slice, t)
		return true
	})

	return slice
}

// Chan generate a channel with a custom bufcap from an iterator.
func Chan[T any](iter Next[T], bufcap int) chan T {
	var channel = make(chan T, bufcap)
	go func() {
		ForEach(iter, func(t T) bool {
			channel <- t
			return true
		})
		close(channel)
	}()

	return channel
}

// KV generate a map from iterator
func KV[M ~map[K]V, K comparable, V any](i Next[Pairs[K, V]]) M {
	var m = make(M)
	ForEach(i, func(p Pairs[K, V]) bool {
		m[p.Key()] = p.Value()
		return true
	})

	return m
}
