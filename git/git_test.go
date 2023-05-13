package git

import (
	"reflect"
	"strings"
	"testing"
)

func Test_extractAuthorSubject(t *testing.T) {
	type args struct {
		patch string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			args: args{
				patch: `From 7e9591868e3985eeeddbfde3cd03901ad6616eef Mon Sep 17 00:00:00 2001
From: Testing <test@email.com>
Date: Thu, 13 Apr 2023 23:38:57 +0200
Subject: [PATCH] test

alalalala
`,
			},
			name:  "basic",
			want:  "Testing <test@email.com>",
			want1: "[PATCH] test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ExtractAuthorSubject(tt.args.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractAuthorSubject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractAuthorSubject() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("extractAuthorSubject() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getRelays(t *testing.T) {
	type args struct {
		relays []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "multiple",
			args: args{
				relays: []string{"relay1", "relay2"},
			},
			want: []string{"relay1", "relay2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Run("config", "nostr.relays", strings.Join(tt.args.relays, " "))
			got, err := GetRelays([]string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("getRelays() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRelays() = %v, want %v", got, tt.want)
			}
		})
	}
}
