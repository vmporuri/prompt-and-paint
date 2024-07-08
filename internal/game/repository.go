package game

import "context"

type roomRepository struct {
	roomList  string
	roomIDKey string
}

var roomRepo = &roomRepository{roomList: string(roomList), roomIDKey: string(roomID)}

func (g *roomRepository) AddRoom(ctx context.Context, roomID string) error {
	return addToRedisSet(ctx, g.roomList, roomID)
}

func (g *roomRepository) DeleteRoom(ctx context.Context, roomID string) error {
	return deleteFromRedisSet(ctx, g.roomList, roomID)
}

func (g *roomRepository) LookupRoom(ctx context.Context, roomID string) (bool, error) {
	return checkMembershipRedisSet(ctx, g.roomList, roomID)
}
