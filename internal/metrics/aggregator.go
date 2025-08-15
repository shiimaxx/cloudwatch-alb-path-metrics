package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/shiimaxx/cloudwatch-alb-path-metrics/internal/alblog"
	"github.com/shiimaxx/cloudwatch-alb-path-metrics/internal/normalizer"
	"github.com/shiimaxx/cloudwatch-alb-path-metrics/pkg/config"
	"github.com/shiimaxx/cloudwatch-alb-path-metrics/pkg/models"
)

type Aggregator struct {
	config     *config.Config
	normalizer *normalizer.PathNormalizer
	parser     *alblog.Parser
}

func NewAggregator(cfg *config.Config) *Aggregator {
	return &Aggregator{
		config:     cfg,
		normalizer: normalizer.NewPathNormalizer(),
		parser:     alblog.NewParser(),
	}
}

func (a *Aggregator) AggregateFromLogData(logData []byte) ([]*models.MetricData, error) {
	entries, err := a.parser.Parse(strings.NewReader(string(logData)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ALB logs: %w", err)
	}
	
	return a.AggregateEntries(entries)
}

func (a *Aggregator) AggregateEntries(entries []*models.ALBLogEntry) ([]*models.MetricData, error) {
	aggregationMap := make(map[models.MetricKey]*models.MetricAggregator)
	
	for _, entry := range entries {
		path, err := a.parser.ExtractPath(entry.RequestURL)
		if err != nil {
			continue
		}
		
		normalizedPath := a.normalizer.Normalize(path)
		
		if !a.config.IsPathAllowed(normalizedPath) {
			continue
		}
		
		minute := entry.Timestamp.Truncate(time.Minute)
		
		key := models.MetricKey{
			Service:         a.config.ServiceName,
			Environment:     a.config.Environment,
			NormalizedRoute: normalizedPath,
			Minute:          minute,
		}
		
		if _, exists := aggregationMap[key]; !exists {
			aggregationMap[key] = &models.MetricAggregator{}
		}
		
		aggregationMap[key].Add(entry)
	}
	
	return a.convertToMetricData(aggregationMap), nil
}

func (a *Aggregator) convertToMetricData(aggregationMap map[models.MetricKey]*models.MetricAggregator) []*models.MetricData {
	var results []*models.MetricData
	
	for key, aggregator := range aggregationMap {
		metricData := &models.MetricData{
			Service:          key.Service,
			Environment:      key.Environment,
			NormalizedRoute:  key.NormalizedRoute,
			Timestamp:        key.Minute,
			RequestsTotal:    aggregator.RequestsTotal,
			RequestsGood:     aggregator.RequestsGood,
			LatencySum:       aggregator.LatencySum,
			LatencyCount:     aggregator.LatencyCount,
		}
		
		results = append(results, metricData)
	}
	
	return results
}