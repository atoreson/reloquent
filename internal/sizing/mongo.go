package sizing

// MongoDB Atlas tier recommendations.
// Migration tier is oversized for throughput; production tier is right-sized.

type mongoTier struct {
	name  string
	ramGB int
	label string
}

var migrationTiers = []struct {
	maxBytes int64
	tier     mongoTier
}{
	{maxBytes: gbToBytes(100), tier: mongoTier{name: "M40", ramGB: 16, label: "M40 (16 GB RAM)"}},
	{maxBytes: gbToBytes(500), tier: mongoTier{name: "M50", ramGB: 32, label: "M50 (32 GB RAM)"}},
	{maxBytes: tbToBytes(2), tier: mongoTier{name: "M60", ramGB: 64, label: "M60 (64 GB RAM)"}},
	{maxBytes: tbToBytes(10), tier: mongoTier{name: "M80", ramGB: 128, label: "M80 (128 GB RAM)"}},
	{maxBytes: tbToBytes(100), tier: mongoTier{name: "M200", ramGB: 256, label: "M200 (256 GB RAM)"}},
}

var productionTiers = []struct {
	maxBytes int64
	tier     mongoTier
}{
	{maxBytes: gbToBytes(50), tier: mongoTier{name: "M30", ramGB: 8, label: "M30 (8 GB RAM)"}},
	{maxBytes: gbToBytes(200), tier: mongoTier{name: "M40", ramGB: 16, label: "M40 (16 GB RAM)"}},
	{maxBytes: tbToBytes(1), tier: mongoTier{name: "M50", ramGB: 32, label: "M50 (32 GB RAM)"}},
	{maxBytes: tbToBytes(5), tier: mongoTier{name: "M60", ramGB: 64, label: "M60 (64 GB RAM)"}},
	{maxBytes: tbToBytes(20), tier: mongoTier{name: "M80", ramGB: 128, label: "M80 (128 GB RAM)"}},
	{maxBytes: tbToBytes(100), tier: mongoTier{name: "M200", ramGB: 256, label: "M200 (256 GB RAM)"}},
}

func calculateMongo(estimatedBytes int64, rowCount int64) MongoPlan {
	// Storage estimate: doc bytes × 1.5 (indexes, padding, overhead)
	storageBytes := int64(float64(estimatedBytes) * 1.5)
	storageGB := ceilInt(bytesToGB(storageBytes))
	if storageGB < 10 {
		storageGB = 10
	}

	// Migration tier (oversized for bulk write throughput)
	migTier := migrationTiers[len(migrationTiers)-1].tier
	for _, t := range migrationTiers {
		if estimatedBytes < t.maxBytes {
			migTier = t.tier
			break
		}
	}

	// Production tier (right-sized for working set)
	prodTier := productionTiers[len(productionTiers)-1].tier
	for _, t := range productionTiers {
		if estimatedBytes < t.maxBytes {
			prodTier = t.tier
			break
		}
	}

	return MongoPlan{
		MigrationTier:   migTier.label,
		ProductionTier:  prodTier.label,
		StorageGB:       int64(storageGB),
		MigrationRAMGB:  migTier.ramGB,
		ProductionRAMGB: prodTier.ramGB,
	}
}
