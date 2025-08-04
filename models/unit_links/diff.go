package unit_links

import (
	"encoding/json"
	"fmt"

	"code.gitea.io/gitea/models/pull_request_sender"
)

type comparableLink struct {
	FromUnitID   int64
	FromUnitType FromUnitType
	ToUnitID     string
	IsActive     bool
	PrStatus     pull_request_sender.FromUnitStatusPr
}

// CalculateDiff сравнивает список существующих линк и новых
func CalculateDiff(old, new AllUnitLinks) (Diff, error) {
	oldLinks := map[comparableLink]struct{}{}

	for _, u := range old {
		link := comparableLink{
			FromUnitID:   u.FromUnitID,
			FromUnitType: u.FromUnitType,
			ToUnitID:     u.ToUnitID,
			IsActive:     u.IsActive,
		}

		oldLinks[link] = struct{}{}
	}

	var addedLinks AllUnitLinks
	for _, u := range new {
		newLink := comparableLink{
			FromUnitID:   u.FromUnitID,
			FromUnitType: u.FromUnitType,
			ToUnitID:     u.ToUnitID,
			IsActive:     u.IsActive,
		}

		if _, isLinkNew := oldLinks[newLink]; isLinkNew {
			delete(oldLinks, newLink)

			continue
		}

		convertedLink := UnitLinks{
			FromUnitID:   u.FromUnitID,
			FromUnitType: u.FromUnitType,
			ToUnitID:     u.ToUnitID,
			IsActive:     u.IsActive,
		}

		addedLinks = append(addedLinks, convertedLink)
		delete(oldLinks, newLink)
	}

	var deletedLinks AllUnitLinks
	for u := range oldLinks {
		convertedLink := UnitLinks{
			FromUnitID:   u.FromUnitID,
			FromUnitType: u.FromUnitType,
			ToUnitID:     u.ToUnitID,
			IsActive:     u.IsActive,
		}

		deletedLinks = append(deletedLinks, convertedLink)
	}

	diff := Diff{
		LinksToAdd:    addedLinks,
		LinksToDelete: deletedLinks,
	}

	return diff, nil
}

func (d Diff) JSON() (string, error) {
	marshalledDiff, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("marshall diff: %w", err)
	}

	return string(marshalledDiff), nil
}
