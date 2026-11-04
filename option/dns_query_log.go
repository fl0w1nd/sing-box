package option

import (
	"github.com/sagernet/sing/common/byteformats"
	"github.com/sagernet/sing/common/json/badoption"
)

type DNSQueryLogOptions struct {
	Enabled       bool                     `json:"enabled,omitempty"`
	Path          string                   `json:"path,omitempty"`
	MemoryOnly    bool                     `json:"memory_only,omitempty"`
	Retention     badoption.Duration       `json:"retention,omitempty"`
	MaxSize       *byteformats.MemoryBytes `json:"max_size,omitempty"`
	MemorySize    int                      `json:"memory_size,omitempty"`
	FlushInterval badoption.Duration       `json:"flush_interval,omitempty"`
}
