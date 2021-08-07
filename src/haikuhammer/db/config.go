package db

import (
	"context"
	"github.com/jonbodner/proteus"
)

type ConfigFlag int64

func (f ConfigFlag) ReactToHaiku() bool {
	return f &ConfigReactToHaiku > 0
}

func (f ConfigFlag) ReactToNonHaiku() bool {
	return f &ConfigReactToNonHaiku > 0
}

func (f ConfigFlag) DeleteNonHaiku() bool {
	return f &ConfigDeleteNonHaiku > 0
}

func (f ConfigFlag) ExplainNonHaiku() bool {
	return f &ConfigExplainNonHaiku > 0
}

func (f ConfigFlag) ServeRandomHaiku() bool {
	return f &ConfigServeRandomHaiku > 0
}

func (f ConfigFlag) Or(other ConfigFlag) ConfigFlag {
	return f | other
}

func (f ConfigFlag) And(other ConfigFlag) ConfigFlag {
	return f & other
}

const (
	ConfigReactToHaiku ConfigFlag = 1 << iota
	ConfigReactToNonHaiku
	ConfigDeleteNonHaiku
	ConfigExplainNonHaiku
	ConfigServeRandomHaiku
)

func LookupFlags(ctx context.Context, e proteus.ContextQuerier, guildID int, channelID int) (ConfigFlag, error) {
	chanConf, err := ChannelConfigDAO.FindByID(ctx, e, channelID)
	if err != nil {
		return 0, err
	}
	guildConf, err := GuildConfigDAO.FindByID(ctx, e, guildID)
	if err != nil {
		return 0, err
	}
	return guildConf.Flags.Or(chanConf.Flags), nil
}

type ChannelConfig struct {
	ChannelID int        `prof:"channel_id"`
	Flags     ConfigFlag `prof:"flags"`
}

var ChannelConfigDAO ChannelConfigDAOImpl

type ChannelConfigDAOImpl struct {
	Upsert func(ctx context.Context, e proteus.ContextExecutor, channelID int, flags int64) (int64, error) `proq:"q:chan_upsert" prop:"channelID,flags"`
	FindByID func(ctx context.Context, e proteus.ContextQuerier, channelID int) (ChannelConfig, error) `proq:"q:chan_findByID" prop:"channelID"`
}

type GuildConfig struct {
	GuildID        int        `prof:"guild_id"`
	Flags          ConfigFlag `prof:"flags"`
	PositiveReacts string     `prof:"positive_reacts"`
	NegativeReacts string     `prof:"negative_reacts"`
}

var GuildConfigDAO GuildConfigDAOImpl

type GuildConfigDAOImpl struct {
	Upsert func(ctx context.Context, e proteus.ContextExecutor, config GuildConfig) (int64, error) `proq:"q:guild_upsert" prop:"config"`
	FindByID func(ctx context.Context, e proteus.ContextQuerier, guildID int) (GuildConfig, error) `proq:"q:guild_findByID" prop:"guildID"`
}

func init() {
	ctx := context.Background()
	m := proteus.MapMapper{
		"chan_upsert": `INSERT INTO channel_config (channel_id, flags)
						VALUES (:channelID:, :flags:)
						ON CONFLICT (channel_id)
						DO UPDATE SET flags = excluded.flags`,
		"chan_findByID": `SELECT * FROM channel_config WHERE channel_id = :channelID:`,
		"guild_upsert": `INSERT INTO guild_config (guild_id, flags, positive_reacts, negative_reacts)
						VALUES (:config.GuildID:, :config.Flags:, :config.PositiveReacts:, :config.NegativeReacts:)
						ON CONFLICT (guild_id)
						DO UPDATE SET flags = excluded.flags, positive_reacts = excluded.positive_reacts, negative_reacts = excluded.negative_reacts`,
		"guild_findByID": `SELECT * FROM guild_config WHERE guild_id = :guildID:`,
	}
	err := proteus.ShouldBuild(ctx, &ChannelConfigDAO, proteus.Sqlite, m)
	if err != nil {
		panic(err)
	}
	err = proteus.ShouldBuild(ctx, &GuildConfigDAO, proteus.Sqlite, m)
	if err != nil {
		panic(err)
	}
}