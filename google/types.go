package google

type SearchResults struct {
	Kind       string
	TotalItems int
	Items      []Volume
}

type Volume struct {
	Kind       string
	Id         string
	Etag       string
	SelfLink   string
	VolumeInfo struct {
		Title               string
		Subtitle            string
		Authors             []string
		Publisher           string
		PublishedDate       string
		Description         string
		IndustryIdentifiers []struct {
			Type       string
			Identifier string
		}
		ReadingModes struct {
			Text  bool
			Image bool
		}
		PageCount           int
		PrintType           string
		Categories          []string
		AverageRating       float64
		RatingsCount        int
		MaturityRating      string
		AllowAnonLogging    bool
		ContentVersion      string
		PanelizationSummary struct {
			ContainsEpubBubbles  bool
			ContainsImageBubbles bool
		}
		ImageLinks struct {
			ExtraLarge     string
			Large          string
			Medium         string
			Small          string
			SmallThumbnail string
			Thumbnail      string
		}
		Language            string
		PreviewLink         string
		InfoLink            string
		CanonicalVolumeLink string
		SeriesInfo          struct {
			BookDisplayNumber    string
			Kind                 string
			ShortSeriesBookTitle string
			Etag                 string
			VolumeSeries         []struct {
				SeriesId       string
				SeriesBookType string
				OrderNumber    int
			}
		}
		SamplePageCount  int
		PrintedPageCount int
		MainCategory     string
		Dimensions       struct {
			Height    int
			Thickness int
			Width     int
		}
		ComicsContent bool
	}
	SaleInfo struct {
		Country     string
		Saleability string
		IsEbook     bool
		ListPrice   struct {
			Amount       float64
			CurrencyCode string
		}
		RetailPrice struct {
			Amount       float64
			CurrencyCode string
		}
		BuyLink string
		Offers  []struct {
			FinskyOfferType int
			ListPrice       struct {
				AmountInMicros int
				CurrencyCode   string
			}
			RetailPrice struct {
				AmountInMicros int
				CurrencyCode   string
			}
		}
		OnSaleDateRaw string
	}
	AccessInfo struct {
		Country                string
		Viewability            string
		Embeddable             bool
		PublicDomain           bool
		TextToSpeechPermission string
		Epub                   struct {
			IsAvailable  bool
			AcsTokenLink string
			DownloadLink string
		}
		Pdf struct {
			IsAvailable  bool
			AcsTokenLink string
			DownloadLink string
		}
		WebReaderLink       string
		AccessViewStatus    string
		QuoteSharingAllowed bool
		DownloadAccess      struct {
			DeviceAllowed      bool
			DownloadsAcquired  int
			JustAcquired       bool
			Kind               string
			MaxDownloadDevices int
			Message            string
			Nonce              string
			ReasonCode         string
			Restricted         bool
			Signature          string
			Source             string
			VolumeId           string
			Etag               string
		}
		DriveImportedContentLink         string
		ExplicitOfflineLicenseManagement bool
		ViewOrderUrl                     string
	}
	SearchInfo struct {
		TextSnippet string
	}
	RecommendedInfo struct {
		Explanation string
	}
	LayerInfo struct {
		Layers []struct {
			LayerId                  int
			VolumeAnnotationsVersion string
		}
	}
}

type SearchParameters struct {
	Download   string
	Filter     string
	StartIndex int
	MaxResults int
	PrintType  string
	Projection string
	OrderBy    string
	Language   string
}
