package rabbitmq

import (
	"sync"
	"sync/atomic"
)

type ChannelProvider struct {
	url     string
	options []ChannelOptionFunc

	mu         sync.Mutex
	channelPtr atomic.Value
}

func NewChannelProvider(url string, options ...ChannelOptionFunc) *ChannelProvider {
	return &ChannelProvider{
		url:     url,
		options: options,
	}
}

func (p *ChannelProvider) GetChannel() (Channel, error) {
	ch := p.channelPtr.Load()
	if ch == nil {
		p.mu.Lock()
		defer p.mu.Unlock()
		ch = p.channelPtr.Load()
		if ch != nil {
			return ch.(Channel), nil
		}
		channel, err := NewChannel(p.url, p.options...)
		if err != nil {
			return nil, err
		}
		p.channelPtr.Store(channel)
		return channel, nil
	} else {
		return ch.(Channel), nil
	}
}

func (p *ChannelProvider) Close() error {
	ch := p.channelPtr.Load()
	if ch != nil {
		channel := ch.(Channel)
		return channel.Close()
	}
	return nil
}

func (p *ChannelProvider) CloseChannel(channel Channel) {
	if channel != nil {
		_ = channel.Close()
		p.channelPtr.CompareAndSwap(channel, nil)
	}
}
