package game

import "context"

// A global room repository used to keep track of all current rooms.
type roomRepository struct {
	roomList  string
	roomIDKey string
}

var roomRepo = &roomRepository{roomList: string(roomList), roomIDKey: string(roomID)}

// Adds a roomID to the global room list.
func (g *roomRepository) addRoom(ctx context.Context, roomID string) error {
	return addToRedisSet(ctx, g.roomList, roomID)
}

// Deletes a roomID from the global room list.
func (g *roomRepository) deleteRoom(ctx context.Context, roomID string) error {
	return deleteFromRedisSet(ctx, g.roomList, roomID)
}

// Looks up whether a room associated with roomID currently exists.
func (g *roomRepository) lookupRoom(ctx context.Context, roomID string) (bool, error) {
	return checkMembershipRedisSet(ctx, g.roomList, roomID)
}
