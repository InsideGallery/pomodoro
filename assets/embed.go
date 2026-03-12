package assets

import "embed"

//go:embed sounds/tick.mp3
var TickSound []byte

//go:embed sounds/alarm.mp3
var AlarmSound []byte

//go:embed fonts/NotoSans-Regular.ttf
var FontRegular []byte

//go:embed fonts/NotoSans-Bold.ttf
var FontBold []byte

//go:embed fonts/*
var Fonts embed.FS

//go:embed sounds/*
var Sounds embed.FS
