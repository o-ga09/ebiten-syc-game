package main

import (
	"encoding/binary"
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

// tone describes one note of a synthesized chiptune sound effect.
type tone struct {
	freq   float64 // Hz (end frequency when slide is set)
	from   float64 // start frequency for a slide, 0 = no slide
	dur    float64 // seconds
	square bool
	noise  bool
}

type Sound struct {
	ctx      *audio.Context
	sounds   map[Event][]byte
	start    []byte
	cursor   []byte
	loudJump []byte
	Loud     bool
	players  []*audio.Player
	Muted    bool
}

func NewSound() *Sound {
	s := &Sound{ctx: audio.NewContext(sampleRate), sounds: map[Event][]byte{}}
	s.sounds[EvJump] = synth(0.10, tone{freq: 990, from: 520, dur: 0.05, square: true})
	s.sounds[EvMiss] = synth(0.22, tone{freq: 70, from: 300, dur: 0.22, noise: true})
	s.sounds[EvItem] = synth(0.15,
		tone{freq: 1047, dur: 0.05, square: true}, tone{freq: 1568, dur: 0.1, square: true})
	s.sounds[EvRare] = synth(0.15,
		tone{freq: 784, dur: 0.06, square: true}, tone{freq: 988, dur: 0.06, square: true},
		tone{freq: 1175, dur: 0.06, square: true}, tone{freq: 1568, dur: 0.2, square: true})
	s.sounds[EvBug] = synth(0.16,
		tone{freq: 220, dur: 0.03, square: true}, tone{freq: 1760, dur: 0.03, square: true},
		tone{freq: 110, dur: 0.03, square: true}, tone{freq: 1320, dur: 0.05, square: true})
	s.sounds[EvOver] = synth(0.18,
		tone{freq: 392, dur: 0.18, square: true}, tone{freq: 330, dur: 0.18, square: true},
		tone{freq: 262, dur: 0.18, square: true}, tone{freq: 196, dur: 0.45, square: true})
	s.sounds[EvReset] = synth(0.16, tone{freq: 1800, from: 120, dur: 0.35, square: true})
	s.loudJump = synth(0.22, tone{freq: 990, from: 520, dur: 0.07, square: true})
	s.start = synth(0.16, tone{freq: 523, dur: 0.06, square: true}, tone{freq: 1047, dur: 0.14, square: true})
	s.cursor = synth(0.12, tone{freq: 660, dur: 0.04, square: true})
	return s
}

func (s *Sound) Play(e Event) {
	if e == EvJump && s.Loud {
		s.play(s.loudJump)
		return
	}
	s.play(s.sounds[e])
}
func (s *Sound) Start()  { s.play(s.start) }
func (s *Sound) Cursor() { s.play(s.cursor) }

func (s *Sound) play(b []byte) {
	if s.Muted || b == nil {
		return
	}
	// Drop finished players to avoid leaking.
	alive := s.players[:0]
	for _, p := range s.players {
		if p.IsPlaying() {
			alive = append(alive, p)
		} else {
			p.PauseAndStopReading()
		}
	}
	s.players = alive
	p := s.ctx.NewPlayerF32FromBytes(b)
	p.Play()
	s.players = append(s.players, p)
}

// synth renders tones sequentially into interleaved stereo float32 PCM.
func synth(volume float64, tones ...tone) []byte {
	var out []byte
	seed := uint32(2463534242)
	for _, t := range tones {
		n := int(t.dur * sampleRate)
		phase := 0.0
		for i := range n {
			p := float64(i) / float64(n)
			f := t.freq
			if t.from > 0 {
				f = t.from + (t.freq-t.from)*p
			}
			phase += f / sampleRate
			var v float64
			switch {
			case t.noise:
				seed ^= seed << 13
				seed ^= seed >> 17
				seed ^= seed << 5
				v = float64(seed)/float64(math.MaxUint32)*2 - 1
			case t.square:
				if math.Mod(phase, 1) < 0.5 {
					v = 1
				} else {
					v = -1
				}
			default:
				v = math.Sin(phase * 2 * math.Pi)
			}
			// Short attack, linear release envelope.
			env := math.Min(1, float64(i)/(sampleRate*0.004)) * (1 - p)
			sample := float32(v * env * volume)
			bits := math.Float32bits(sample)
			out = binary.LittleEndian.AppendUint32(out, bits)
			out = binary.LittleEndian.AppendUint32(out, bits)
		}
	}
	return out
}
