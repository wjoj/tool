package metric

type (
	Metric interface {
		Add(int642 int64)
		Value() int64
	}

	Aggregation interface {
		Sum() float64
		Min() float64
		Avg() float64
		Max() float64
	}

	// VectorOpts contains the common arguments for creating vec Metric..
	VectorOpts struct {
		Namespace string
		Subsystem string
		Name      string
		Help      string
		Labels    []string
	}
)

const (
	_businessNamespace          = "business"
	_businessSubsystemCount     = "count"
	_businessSubSystemGauge     = "gauge"
	_businessSubSystemHistogram = "histogram"
)
