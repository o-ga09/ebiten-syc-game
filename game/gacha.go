package main

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
)

// Birth is the result of the birth gacha.
type Birth struct {
	Talents    []TalentID
	Difficulty Difficulty
	Family     Family
	Look       Look
}

func (b Birth) Has(id TalentID) bool { return slices.Contains(b.Talents, id) }

// LookKey is a stable string representation of the appearance.
func (b Birth) LookKey() string {
	return fmt.Sprintf("h%d-c%d-x%d", b.Look.Hair, b.Look.Clothes, b.Look.Hat)
}

// ComboKey identifies the gacha combination for the "one and only" title.
func (b Birth) ComboKey() string {
	ids := make([]string, len(b.Talents))
	for i, t := range b.Talents {
		ids[i] = string(t)
	}
	slices.Sort(ids)
	return strings.Join(ids, ",") + "|" + string(b.Difficulty.ID) + "|" + string(b.Family.ID)
}

func RollBirth(rng *rand.Rand) Birth {
	var b Birth
	for len(b.Talents) < 3 {
		t := rollTalent(rng)
		if !b.Has(t.ID) {
			b.Talents = append(b.Talents, t.ID)
		}
	}
	b.Difficulty = rollDifficulty(rng)
	b.Family = families[rng.IntN(len(families))]
	b.Look = Look{
		Hair:    rng.IntN(len(hairColors)),
		Hat:     rng.IntN(len(hatNames)),
		Clothes: rng.IntN(len(clothesColors)),
	}
	return b
}

func rollTalent(rng *rand.Rand) Talent {
	total := 0
	for _, t := range talents {
		total += rarityWeights[t.Rarity]
	}
	r := rng.IntN(total)
	for _, t := range talents {
		r -= rarityWeights[t.Rarity]
		if r < 0 {
			return t
		}
	}
	return talents[len(talents)-1]
}

func rollDifficulty(rng *rand.Rand) Difficulty {
	total := 0
	for _, d := range difficulties {
		total += d.Weight
	}
	r := rng.IntN(total)
	for _, d := range difficulties {
		r -= d.Weight
		if r < 0 {
			return d
		}
	}
	return difficulties[1]
}
