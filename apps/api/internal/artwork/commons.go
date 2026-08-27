package artwork

import (
	"context"
	"strings"
)

// CommonsSource offers Wikimedia Commons images.
//
// These are freely licensed, which is the point: they can be re-hosted and
// credited without ambiguity. The trade is subject matter — Commons has far
// more photographs of games being played than pictures of boxes — so this sits
// below the publisher source in priority rather than above it.
type CommonsSource struct {
	Facts map[int]Facts
}

func (s *CommonsSource) Name() string { return "commons" }

func (s *CommonsSource) Find(_ context.Context, g Game) (*Candidate, error) {
	f, ok := s.Facts[g.BGGID]
	if !ok || f.CommonsImage == "" {
		return nil, nil
	}

	// Special:FilePath redirects to the file itself and accepts a width, so ask
	// for something a card can use rather than the full-resolution original.
	src := f.CommonsImage
	if strings.Contains(src, "Special:FilePath") {
		src += "?width=800"
	}

	return &Candidate{
		URL:     src,
		Source:  "commons",
		License: "See Wikimedia Commons file page",
		Credit:  "Wikimedia Commons",
		Origin:  f.CommonsImage,
	}, nil
}
