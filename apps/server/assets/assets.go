package assets

import "embed"

// FS contains the server-owned static files and email templates.
//
//go:embed static/* templates/emails/*
var FS embed.FS
