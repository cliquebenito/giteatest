//go:build !correct

// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package sbt_binding_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitea.com/go-chi/binding"
	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

const (
	testRoute    = "/test"
	maxSizeCount = 9
	minSizeCount = 4
)

type (
	validationTestCase struct {
		description    string
		data           interface{}
		expectedErrors binding.Errors
	}

	SbtTestForm struct {
		Email        string `form:"Email" binding:"SbtEmail"`
		MaxSize      string `form:"MaxSize" binding:"SbtMaxSize(9)"`
		MinSize      string `form:"MinSize" binding:"SbtMinSize(4)"`
		Url          string `form:"Url" binding:"SbtUrl"`
		In           string `form:"In" binding:"SbtIn(public,limited,private)"`
		GitRefName   string `form:"GitRefName" binding:"SbtGitRefName"`
		AlphaDashDot string `form:"AlphaDashDot" binding:"SbtAlphaDashDot"`
		NotEmpty     string `form:"NotEmpty" binding:"SbtNotEmpty"`
		Range        int    `form:"Range" binding:"SbtRange(4,9)"`
	}

	SbtTestFormOptional struct {
		Email        *string `form:"Email" binding:"SbtEmail"`
		MaxSize      *string `form:"MaxSize" binding:"SbtMaxSize(9)"`
		MinSize      *string `form:"MinSize" binding:"SbtMinSize(4)"`
		Url          *string `form:"Url" binding:"SbtUrl"`
		In           *string `form:"In" binding:"SbtIn(public,limited,private)"`
		GitRefName   *string `form:"GitRefName" binding:"SbtGitRefName"`
		AlphaDashDot *string `form:"AlphaDashDot" binding:"SbtAlphaDashDot"`
		NotEmpty     *string `form:"NotEmpty" binding:"SbtNotEmpty"`
		Range        *int    `form:"Range" binding:"SbtRange(4,9)"`
	}
)

func validationTest(t *testing.T, testCase validationTestCase) {
	httpRecorder := httptest.NewRecorder()
	m := chi.NewRouter()

	m.Post(testRoute, func(resp http.ResponseWriter, req *http.Request) {
		actual := binding.Validate(req, testCase.data)
		if actual == nil {
			actual = binding.Errors{}
		}

		assert.Equal(t, testCase.expectedErrors, actual)
	})

	req, err := http.NewRequest("POST", testRoute, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "x-www-form-urlencoded")
	m.ServeHTTP(httpRecorder, req)

	switch httpRecorder.Code {
	case http.StatusNotFound:
		panic("Routing is messed up in test fixture (got 404): check methods and paths")
	case http.StatusInternalServerError:
		panic("Something bad happened on '" + testCase.description + "'")
	}
}
