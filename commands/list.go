package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ambientsound/pms/list"
	"github.com/ambientsound/pms/log"
	"github.com/ambientsound/pms/spotify/devices"
	"github.com/ambientsound/pms/spotify/library"
	"github.com/ambientsound/pms/spotify/playlists"
	"github.com/ambientsound/pms/spotify/tracklist"
	"github.com/zmb3/spotify"

	"github.com/ambientsound/pms/api"
	"github.com/ambientsound/pms/input/lexer"
)

// List navigates and manipulates songlists.
type List struct {
	command
	api       api.API
	client    *spotify.Client
	absolute  int
	duplicate bool
	goto_     bool
	open      bool
	relative  int
	remove    bool
	name      string
}

func NewList(api api.API) Command {
	return &List{
		api:      api,
		absolute: -1,
	}
}

func (cmd *List) Parse() error {
	tok, lit := cmd.ScanIgnoreWhitespace()
	cmd.setTabCompleteVerbs(lit)

	switch tok {
	case lexer.TokenIdentifier:
		switch lit {
		case "duplicate":
			cmd.duplicate = true
		case "remove":
			cmd.remove = true
		case "up", "prev", "previous":
			cmd.relative = -1
		case "down", "next":
			cmd.relative = 1
		case "home":
			cmd.absolute = 0
		case "end":
			cmd.absolute = cmd.api.Db().Len() - 1
		case "goto":
			cmd.goto_ = true
		case "open":
			cmd.open = true
		default:
			i, err := strconv.Atoi(lit)
			if err != nil {
				return fmt.Errorf("cannot navigate lists: position '%s' is not recognized, and is not a number", lit)
			}
			cmd.absolute = i - 1
		}
	default:
		return fmt.Errorf("unexpected '%s', expected identifier", lit)
	}

	if cmd.goto_ {
		for tok != lexer.TokenEnd {
			tok, lit = cmd.ScanIgnoreWhitespace()
			cmd.name += lit
		}

		cmd.Unscan()
		cmd.setTabComplete(cmd.name, cmd.api.Db().Keys())
	} else {
		cmd.setTabCompleteEmpty()
	}

	return cmd.ParseEnd()
}

func (cmd *List) Exec() error {
	switch {
	case cmd.goto_:
		return cmd.Goto(cmd.name)

	case cmd.open:
		row := cmd.api.List().CursorRow()
		if row == nil {
			return fmt.Errorf("no playlist selected")
		}
		return cmd.Goto(row[list.RowIDKey])

	case cmd.relative != 0:
		cmd.api.Db().MoveCursor(cmd.relative)
		cmd.api.SetList(cmd.api.Db().Current())

	case cmd.absolute >= 0:
		cmd.api.Db().SetCursor(cmd.absolute)
		cmd.api.SetList(cmd.api.Db().Current())

	case cmd.duplicate:
		tracklist := cmd.api.Tracklist()
		if tracklist == nil {
			return fmt.Errorf("only track lists can be duplicated")
		}
		return fmt.Errorf("duplicate is not implemented")

	case cmd.remove:
		return fmt.Errorf("remove is not implemented")
	}

	return nil
}

// Goto loads an external list and applies default columns and sorting.
// Local, cached versions are tried first.
func (cmd *List) Goto(id string) error {
	var err error
	var lst list.List

	// Set Spotify object request limit. Ignore user-defined max limit here,
	// because big queries will always be faster and consume less bandwidth,
	// when requesting all the data.
	const limit = 50

	// Try a cached version of a named list
	lst = cmd.api.Db().List(cmd.name)
	if lst != nil {
		cmd.api.SetList(lst)
		return nil
	}

	// Other named lists need Spotify access
	cmd.client, err = cmd.api.Spotify()
	if err != nil {
		return err
	}

	t := time.Now()
	switch id {
	case spotify_library.MyPlaylists:
		lst, err = cmd.gotoMyPrivatePlaylists(limit)
	case spotify_library.FeaturedPlaylists:
		lst, err = cmd.gotoFeaturedPlaylists(limit)
	case spotify_library.MyTracks:
		lst, err = cmd.gotoMyTracks(limit)
	case spotify_library.TopTracks:
		lst, err = cmd.gotoTopTracks(limit)
	case spotify_library.Devices:
		lst, err = cmd.gotoDevices()
	default:
		lst, err = cmd.gotoListWithID(id, limit)
	}
	dur := time.Since(t)

	if err != nil {
		return err
	}

	log.Debugf("Retrieved %s with %d items in %s", id, lst.Len(), dur.String())
	log.Infof("Loaded %s.", lst.Name())

	// Reset cursor
	lst.SetCursor(0)

	cmd.api.SetList(lst)

	return nil
}

func (cmd *List) gotoListWithID(id string, limit int) (list.List, error) {
	sid := spotify.ID(id)

	playlist, err := cmd.client.GetPlaylist(sid)
	if err != nil {
		return nil, err
	}

	tracks, err := cmd.client.GetPlaylistTracksOpt(sid, &spotify.Options{
		Limit: &limit,
	}, "")
	if err != nil {
		return nil, err
	}

	lst, err := spotify_tracklist.NewFromPlaylistTrackPage(*cmd.client, tracks)
	if err != nil {
		return nil, err
	}

	lst.SetName(fmt.Sprintf("%s by %s", playlist.Name, playlist.Owner.DisplayName))
	lst.SetID(id)
	cmd.defaultColumns(lst)

	return lst, nil
}

func (cmd *List) gotoMyPrivatePlaylists(limit int) (list.List, error) {
	playlists, err := cmd.client.CurrentUsersPlaylistsOpt(&spotify.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}

	lst, err := spotify_playlists.New(*cmd.client, playlists)
	if err != nil {
		return nil, err
	}

	lst.SetName("My playlists")
	lst.SetID(spotify_library.MyPlaylists)
	lst.SetVisibleColumns(lst.ColumnNames())

	return lst, nil
}

func (cmd *List) gotoFeaturedPlaylists(limit int) (list.List, error) {
	message, playlists, err := cmd.client.FeaturedPlaylistsOpt(&spotify.PlaylistOptions{
	})
	if err != nil {
		return nil, err
	}

	lst, err := spotify_playlists.New(*cmd.client, playlists)
	if err != nil {
		return nil, err
	}

	lst.SetName(message)
	lst.SetID(spotify_library.FeaturedPlaylists)
	lst.SetVisibleColumns(lst.ColumnNames())

	return lst, nil
}

func (cmd *List) gotoMyTracks(limit int) (list.List, error) {
	tracks, err := cmd.client.CurrentUsersTracksOpt(&spotify.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}

	lst, err := spotify_tracklist.NewFromSavedTrackPage(*cmd.client, tracks)
	if err != nil {
		return nil, err
	}

	lst.SetName("Saved tracks")
	lst.SetID(spotify_library.MyTracks)
	cmd.defaultSort(lst)
	cmd.defaultColumns(lst)

	return lst, nil
}

func (cmd *List) gotoTopTracks(limit int) (list.List, error) {
	tracks, err := cmd.client.CurrentUsersTopTracksOpt(&spotify.Options{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}

	lst, err := spotify_tracklist.NewFromFullTrackPage(*cmd.client, tracks)
	if err != nil {
		return nil, err
	}

	lst.SetName("Top tracks")
	lst.SetID(spotify_library.TopTracks)
	cmd.defaultColumns(lst)

	return lst, nil
}

func (cmd *List) gotoDevices() (list.List, error) {
	return spotify_devices.New(*cmd.client)
}

// setTabCompleteVerbs sets the tab complete list to the list of available sub-commands.
func (cmd *List) setTabCompleteVerbs(lit string) {
	cmd.setTabComplete(lit, []string{
		"down",
		"duplicate",
		"end",
		"goto",
		"home",
		"next",
		"prev",
		"previous",
		"remove",
		"up",
	})
}

// Apply default sorting to a list
func (cmd *List) defaultSort(lst list.List) {
	sort := strings.Split(cmd.api.Options().GetString("sort"), ",")
	_ = lst.Sort(sort)
}

// Show default columns for all named lists
func (cmd *List) defaultColumns(lst list.List) {
	cols := strings.Split(cmd.api.Options().GetString("columns"), ",")
	lst.SetVisibleColumns(cols)
}
