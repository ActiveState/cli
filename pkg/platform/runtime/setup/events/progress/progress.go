package progress

type Reporter interface {
	ReportSize(int) error
	ReportIncrement(int) error
}

type Report struct {
	ReportSizeCb      func(int) error
	ReportIncrementCb func(int) error
}

func (p *Report) ReportSize(size int) error {
	return p.ReportSizeCb(size)
}

func (p *Report) ReportIncrement(inc int) error {
	return p.ReportIncrementCb(inc)
}
