package tool

type Options struct {
	quit bool
}

type Option func(c *Options)

func WithQuitEnableOption() Option {
	return func(c *Options) {
		c.quit = true
	}
}

func applyOptions(options ...Option) Options {
	opts := Options{
		quit: false,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&opts)
	}
	return opts
}
