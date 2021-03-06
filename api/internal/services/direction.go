package services

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/ythosa/rating-list-monitoring-platform-api/internal/dto"
	"github.com/ythosa/rating-list-monitoring-platform-api/internal/models"
	"github.com/ythosa/rating-list-monitoring-platform-api/internal/repository"
	"github.com/ythosa/rating-list-monitoring-platform-api/internal/repository/rdto"
	"github.com/ythosa/rating-list-monitoring-platform-api/pkg/logging"
)

type DirectionImpl struct {
	directionRepository repository.Direction
	userRepository      repository.User
	universityService   University
	parsingService      Parsing
	logger              *logging.Logger
}

func NewDirectionImpl(
	directionRepository repository.Direction,
	userRepository repository.User,
	universityService University,
	parsingService Parsing,
) *DirectionImpl {
	return &DirectionImpl{
		directionRepository: directionRepository,
		userRepository:      userRepository,
		universityService:   universityService,
		parsingService:      parsingService,
		logger:              logging.NewLogger("directions services"),
	}
}

func (s *DirectionImpl) GetByID(id uint) (*models.Direction, error) {
	direction, err := s.directionRepository.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("error while getting direction by id by repository: %w", err)
	}

	return direction, nil
}

func (s *DirectionImpl) GetAll() ([]dto.UniversityDirections, error) {
	directions, err := s.directionRepository.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error while getting all directions by repository: %w", err)
	}

	return s.mapDirectionsToUniversityDirections(directions), nil
}

func (s *DirectionImpl) GetForUser(userID uint) ([]dto.UniversityDirections, error) {
	directions, err := s.directionRepository.GetForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("error while getting user directions by repository: %w", err)
	}

	universityDirections := s.mapDirectionsToUniversityDirections(directions)
	sortUniversityDirections(universityDirections)

	return universityDirections, nil
}

func (s *DirectionImpl) mapDirectionsToUniversityDirections(directions []rdto.Direction) []dto.UniversityDirections {
	ud := make(map[uint]*dto.UniversityDirections)

	for _, d := range directions {
		if _, ok := ud[d.UniversityID]; !ok {
			ud[d.UniversityID] = &dto.UniversityDirections{
				UniversityID:       d.UniversityID,
				UniversityName:     d.UniversityName,
				UniversityFullName: d.UniversityFullName,
				Directions:         make([]dto.Direction, 0),
			}
		}

		ud[d.UniversityID].Directions = append(ud[d.UniversityID].Directions, dto.Direction{
			ID:   d.DirectionID,
			Name: d.DirectionName,
		})
	}

	universityDirections := make([]dto.UniversityDirections, 0)
	for _, v := range ud {
		universityDirections = append(universityDirections, *v)
	}

	return universityDirections
}

func sortUniversityDirections(universityDirections []dto.UniversityDirections) {
	sort.SliceStable(universityDirections, func(i, j int) bool {
		return universityDirections[i].UniversityID < universityDirections[j].UniversityID
	})

	for _, ud := range universityDirections {
		ud.SortDirections()
	}
}

type parsingDirectionResults struct {
	directionsWithRating chan dto.DirectionWithParsingResult
	errors               chan error
	wg                   *sync.WaitGroup
}

func newParsingDirectionResults(directionsCount int) *parsingDirectionResults {
	return &parsingDirectionResults{
		directionsWithRating: make(chan dto.DirectionWithParsingResult, directionsCount),
		errors:               make(chan error, directionsCount),
		wg:                   &sync.WaitGroup{},
	}
}

func (s *DirectionImpl) GetForUserWithRating(userID uint) ([]dto.UniversityDirectionsWithRating, error) {
	directions, err := s.directionRepository.GetForUser(userID)
	if err != nil {
		s.logger.Error(err)

		return nil, fmt.Errorf("error while getting user directions by repository: %w", err)
	}

	userSnils, err := s.userRepository.GetSnils(userID)
	if err != nil {
		s.logger.Error(err)

		return nil, fmt.Errorf("error while getting user snils by repository: %w", err)
	}

	results := newParsingDirectionResults(len(directions))
	for _, d := range directions {
		results.wg.Add(1)
		go s.parseDirectionRating(results, d, userSnils.Snils)
	}

	results.wg.Wait()
	if len(results.errors) != 0 {
		return nil, fmt.Errorf("error while waiting parsing rating: %w", <-results.errors)
	}

	directionsWithRating := make([]dto.DirectionWithParsingResult, len(directions))
	for i := 0; i < len(directions); i++ {
		directionsWithRating[i] = <-results.directionsWithRating
	}

	universityDirectionsWithRating := s.mapRatingDirectionsToUniversityDirections(directionsWithRating)
	sortUniversityDirectionsWithRating(universityDirectionsWithRating)

	return universityDirectionsWithRating, nil
}

func (s *DirectionImpl) parseDirectionRating(
	results *parsingDirectionResults,
	direction rdto.Direction,
	userSnils string,
) {
	defer results.wg.Done()

	parsingResult, err := s.parsingService.ParseRating(
		direction.UniversityName,
		direction.DirectionURL,
		userSnils,
	)
	if err != nil && errors.Is(err, ErrUserNotFoundInRatingList) {
		parsingResult = &dto.EmptyParsingResult
	} else if err != nil {
		results.errors <- fmt.Errorf("error while parsong rating list: %w", err)

		return
	}

	results.directionsWithRating <- dto.DirectionWithParsingResult{
		Direction:     direction,
		ParsingResult: *parsingResult,
	}
}

func (s *DirectionImpl) mapRatingDirectionsToUniversityDirections(
	directions []dto.DirectionWithParsingResult,
) []dto.UniversityDirectionsWithRating {
	ud := make(map[uint]*dto.UniversityDirectionsWithRating)

	for _, d := range directions {
		universityID := d.Direction.UniversityID
		if _, ok := ud[universityID]; !ok {
			ud[d.Direction.UniversityID] = &dto.UniversityDirectionsWithRating{
				UniversityID:       universityID,
				UniversityName:     d.Direction.UniversityName,
				UniversityFullName: d.Direction.UniversityFullName,
				Directions:         make([]dto.DirectionWithRating, 0),
			}
		}

		ud[universityID].Directions = append(ud[universityID].Directions, dto.NewDirectionWithRating(d))
	}

	universityDirections := make([]dto.UniversityDirectionsWithRating, 0)
	for _, v := range ud {
		universityDirections = append(universityDirections, *v)
	}

	return universityDirections
}

func sortUniversityDirectionsWithRating(universityDirections []dto.UniversityDirectionsWithRating) {
	sort.SliceStable(universityDirections, func(i, j int) bool {
		return universityDirections[i].UniversityID < universityDirections[j].UniversityID
	})

	for _, ud := range universityDirections {
		ud.SortDirections()
	}
}

func (s *DirectionImpl) SetForUser(userID uint, directionIDs dto.IDs) error {
	if err := s.directionRepository.Clear(userID); err != nil {
		return fmt.Errorf("error while clearing user directions by repository: %w", err)
	}

	if err := s.directionRepository.SetForUser(userID, directionIDs); err != nil {
		return fmt.Errorf("error while setting directions for user by repository: %w", err)
	}

	if err := s.universityService.SetForUser(
		userID, dto.IDs{IDs: s.getUniversityIDsOfDirections(directionIDs.IDs)},
	); err != nil {
		return fmt.Errorf("error while updating user universities by repository: %w", err)
	}

	return nil
}

func (s *DirectionImpl) getUniversityIDsOfDirections(directionIDs []uint) []uint {
	var (
		universityIDsMap sync.Map
		wg               sync.WaitGroup
	)

	for _, directionID := range directionIDs {
		wg.Add(1)

		go func(id uint) {
			d, _ := s.directionRepository.GetUniversityID(id)
			universityIDsMap.Store(d.UniversityID, true)
			wg.Done()
		}(directionID)
	}

	wg.Wait()

	universityIDs := make([]uint, 0)
	universityIDsMap.Range(func(key, _ interface{}) bool {
		universityIDs = append(universityIDs, key.(uint))

		return true
	})

	return universityIDs
}
