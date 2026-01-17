package classify

import "github.com/RajarshiParmar/groundtruth/internal/model"

type Classifier interface {
	Classify(commit model.Commit) model.ClassifiedCommit
}
