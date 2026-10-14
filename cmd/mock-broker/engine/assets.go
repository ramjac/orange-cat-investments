package engine

// AssetConfig defines parameters for an equity asset.
type AssetConfig struct {
	Symbol          string
	Name            string
	AssetID         string
	Exchange        string
	InitialPrice    float64
	AnnualDrift     float64 // mu
	Volatility      float64 // sigma
	SpreadBps       float64
	AvgVolumePerSec int64
}

// DefaultAssets returns the initial universe of equities for the mock broker.
func DefaultAssets() []AssetConfig {
	return []AssetConfig{
		{
			Symbol:          "AAPL",
			Name:            "Apple Inc.",
			AssetID:         "b0b6dd9d-8b9b-48a9-ba46-b9d54906e401",
			Exchange:        "NASDAQ",
			InitialPrice:    224.30,
			AnnualDrift:     0.09,
			Volatility:      0.24,
			SpreadBps:       0.0004,
			AvgVolumePerSec: 1500,
		},
		{
			Symbol:          "NVDA",
			Name:            "NVIDIA Corporation",
			AssetID:         "b0b6dd9d-8b9b-48a9-ba46-b9d54906e402",
			Exchange:        "NASDAQ",
			InitialPrice:    138.80,
			AnnualDrift:     0.18,
			Volatility:      0.44,
			SpreadBps:       0.0005,
			AvgVolumePerSec: 3200,
		},
		{
			Symbol:          "MSFT",
			Name:            "Microsoft Corporation",
			AssetID:         "b0b6dd9d-8b9b-48a9-ba46-b9d54906e403",
			Exchange:        "NASDAQ",
			InitialPrice:    448.20,
			AnnualDrift:     0.08,
			Volatility:      0.21,
			SpreadBps:       0.0003,
			AvgVolumePerSec: 800,
		},
		{
			Symbol:          "TSLA",
			Name:            "Tesla Inc.",
			AssetID:         "b0b6dd9d-8b9b-48a9-ba46-b9d54906e404",
			Exchange:        "NASDAQ",
			InitialPrice:    247.12,
			AnnualDrift:     0.11,
			Volatility:      0.54,
			SpreadBps:       0.0008,
			AvgVolumePerSec: 1800,
		},
		{
			Symbol:          "SPY",
			Name:            "SPDR S&P 500 ETF Trust",
			AssetID:         "b0b6dd9d-8b9b-48a9-ba46-b9d54906e405",
			Exchange:        "NYSE",
			InitialPrice:    585.00,
			AnnualDrift:     0.07,
			Volatility:      0.15,
			SpreadBps:       0.0001,
			AvgVolumePerSec: 4000,
		},
	}
}
