package inspect

import "github.com/nucleuskit/contract/manifest"

func freshness(dir string, m manifest.Manifest) []GeneratedFreshness {
	if len(m.AI.Generated) == 0 {
		return nil
	}
	sourceHash, err := ContractSourceHash(dir)
	if err != nil {
		sourceHash = ""
	}
	items := make([]GeneratedFreshness, 0, len(m.AI.Generated))
	for _, target := range m.AI.Generated {
		item := GeneratedFreshness{
			Source:     "api",
			Target:     target,
			SourceHash: sourceHash,
		}
		targetHash, err := ReadGeneratedHash(dir, target)
		if err != nil {
			item.Reason = "missing freshness marker"
			items = append(items, item)
			continue
		}
		item.TargetHash = targetHash
		item.Fresh = sourceHash != "" && targetHash == sourceHash
		if !item.Fresh {
			item.Reason = "source hash differs from generated marker"
		}
		items = append(items, item)
	}
	return items
}
