package logs

import (
	"sync"

	ilogs "golan-example/internal/logs"
)

var configureOnce sync.Once

// Option 是 Configure 的設定選項。
type Option func(*config)

type config struct {
	global levelConfig
	levels [4]*levelConfig
}

type levelConfig struct {
	filters   []ilogs.Handler
	enrichers []ilogs.Handler
	pipes     []pipeConfig
	noCaller  bool
	noInherit bool
}

type pipeConfig struct {
	formatter ilogs.Formatter
	output    Output
}

// Configure 設定 logging 輸出行為。sync.Once 保護，僅首次呼叫生效。
func Configure(opts ...Option) {
	configureOnce.Do(func() {
		c := &config{}
		for _, o := range opts {
			o(c)
		}
		chains := buildChains(c)
		ilogs.Init(chains)
	})
}

func WithFilter(filters ...Filter) Option {
	return func(c *config) {
		c.global.filters = append(c.global.filters, filters...)
	}
}

func WithEnrichment(enrichers ...Enricher) Option {
	return func(c *config) {
		c.global.enrichers = append(c.global.enrichers, enrichers...)
	}
}

func Pipe(f Formatter, o Output) Option {
	return func(c *config) {
		c.global.pipes = append(c.global.pipes, pipeConfig{formatter: f, output: o})
	}
}

func NoCaller() Option {
	return func(c *config) {
		c.global.noCaller = true
	}
}

func NoInherit() Option {
	return func(c *config) {
		c.global.noInherit = true
	}
}

func ForDebug(opts ...Option) Option { return forLevel(ilogs.LevelDebug, opts) }
func ForInfo(opts ...Option) Option  { return forLevel(ilogs.LevelInfo, opts) }
func ForWarn(opts ...Option) Option  { return forLevel(ilogs.LevelWarn, opts) }
func ForError(opts ...Option) Option { return forLevel(ilogs.LevelError, opts) }

func forLevel(level ilogs.Level, opts []Option) Option {
	return func(c *config) {
		if c.levels[level] == nil {
			c.levels[level] = &levelConfig{}
		}
		sub := &config{}
		for _, o := range opts {
			o(sub)
		}
		lc := c.levels[level]
		lc.filters = append(lc.filters, sub.global.filters...)
		lc.enrichers = append(lc.enrichers, sub.global.enrichers...)
		lc.pipes = append(lc.pipes, sub.global.pipes...)
		if sub.global.noCaller {
			lc.noCaller = true
		}
		if sub.global.noInherit {
			lc.noInherit = true
		}
	}
}

func buildChains(c *config) [4]*ilogs.Chain {
	var chains [4]*ilogs.Chain
	for i := 0; i < 4; i++ {
		level := ilogs.Level(i)
		lc := c.levels[i]

		var merged levelConfig
		if lc != nil && lc.noInherit {
			merged = *lc
		} else {
			merged = mergeConfigs(c.global, lc)
		}

		var handlers []ilogs.Handler
		handlers = append(handlers, merged.filters...)

		// Caller: default enabled unless noCaller
		if !merged.noCaller {
			handlers = append(handlers, &ilogs.CallerEnricher{})
		}

		handlers = append(handlers, merged.enrichers...)

		if len(merged.pipes) == 0 {
			continue
		}

		sinks := make([]ilogs.SinkWriter, 0, len(merged.pipes))
		for _, p := range merged.pipes {
			// Set ext for fileOutput before Resolve
			if fo, ok := p.output.(*fileOutput); ok {
				fo.ext = formatterExt(p.formatter)
			}
			writer := p.output.Resolve(ilogs.ResolveContext{Level: level})
			sinks = append(sinks, ilogs.NewSink(p.formatter, writer))
		}

		chains[i] = ilogs.NewChain(handlers, ilogs.FanOut(sinks))
	}
	return chains
}

func mergeConfigs(global levelConfig, level *levelConfig) levelConfig {
	merged := levelConfig{
		filters:   append([]ilogs.Handler{}, global.filters...),
		enrichers: append([]ilogs.Handler{}, global.enrichers...),
		pipes:     append([]pipeConfig{}, global.pipes...),
		noCaller:  global.noCaller,
	}
	if level != nil {
		merged.filters = append(merged.filters, level.filters...)
		merged.enrichers = append(merged.enrichers, level.enrichers...)
		merged.pipes = append(merged.pipes, level.pipes...)
		if level.noCaller {
			merged.noCaller = true
		}
	}
	return merged
}
