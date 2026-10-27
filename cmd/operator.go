// cmd/operator.go

package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"smegg.me/qr2g.it/common/messages"
	"smegg.me/qr2g.it/core/automation"
	"smegg.me/qr2g.it/core/datastore"
	"smegg.me/qr2g.it/core/models"
	"smegg.me/qr2g.it/core/repositories"
	"smegg.me/qr2g.it/core/slug"
)

func newTokenCmd() *cobra.Command {
	token := &cobra.Command{
		Use:   "token",
		Short: messages.CliTokenShort(say),
		Args:  usage(cobra.NoArgs),
	}
	var name string
	var quota int
	create := &cobra.Command{
		Use:   "create",
		Short: messages.CliTokenCreate(say),
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := openStore(cmd.Context()); err != nil {
				return err
			}
			svc := automation.NewService(repositories.NewAutomationRepository(datastore.Get()))
			raw, t, err := svc.Issue(cmd.Context(), name, quota)
			if err != nil {
				return err
			}
			printTable(stdout(cmd), []string{messages.CliColumnId(say), messages.CliColumnName(say), messages.CliColumnQuota(say)},
				[][]string{{strconv.FormatInt(t.ID, 10), t.Name, strconv.Itoa(t.DailyQuota)}})
			_, err = fmt.Fprintln(stdout(cmd), raw)
			return err
		},
	}
	create.Flags().StringVar(&name, "name", "", messages.CliTokenNameFlag(say))
	create.Flags().IntVar(&quota, "quota", 1000, messages.CliTokenQuotaFlag(say))
	_ = create.MarkFlagRequired("name")

	list := &cobra.Command{
		Use:   "list",
		Short: messages.CliTokenList(say),
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := openStore(cmd.Context()); err != nil {
				return err
			}
			tokens, err := automation.NewService(repositories.NewAutomationRepository(datastore.Get())).List(cmd.Context())
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(tokens))
			for _, t := range tokens {
				rows = append(rows, []string{strconv.FormatInt(t.ID, 10), t.Name, strconv.Itoa(t.DailyQuota),
					t.CreatedAt.UTC().Format(time.DateOnly), when(t.LastUsedAt), when(t.RevokedAt)})
			}
			printTable(stdout(cmd), []string{messages.CliColumnId(say), messages.CliColumnName(say), messages.CliColumnQuota(say),
				messages.CliColumnCreated(say), messages.CliColumnLastUsed(say), messages.CliColumnRevoked(say)}, rows)
			return nil
		},
	}

	revoke := &cobra.Command{
		Use:   "revoke ID",
		Short: messages.CliTokenRevoke(say),
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("%w: %s", errUsage, messages.CliTokenIdNumber(say))
			}
			if err := openStore(cmd.Context()); err != nil {
				return err
			}
			err = automation.NewService(repositories.NewAutomationRepository(datastore.Get())).Revoke(cmd.Context(), id)
			if errors.Is(err, repositories.ErrNotFound) {
				return errors.New(messages.CliTokenMissing(say, messages.CliTokenMissingParams{Id: args[0]}))
			}
			return err
		},
	}
	token.AddCommand(create, list, revoke)
	return token
}

func when(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.UTC().Format(time.DateTime)
}

func newModerateCmd() *cobra.Command {
	moderate := &cobra.Command{
		Use:   "moderate",
		Short: messages.CliModerateShort(say),
		Args:  usage(cobra.NoArgs),
	}
	var note string
	set := func(action models.Moderation, use, short string) *cobra.Command {
		c := &cobra.Command{
			Use:   use + " SLUG",
			Short: short,
			Args:  usage(cobra.ExactArgs(1)),
			RunE: func(cmd *cobra.Command, args []string) error {
				id, err := slug.Decode(args[0])
				if err != nil {
					return fmt.Errorf("%w: %s", errUsage, messages.CliModerateNotSlug(say, messages.CliModerateNotSlugParams{Value: args[0]}))
				}
				if err := openStore(cmd.Context()); err != nil {
					return err
				}
				links := repositories.NewLinkRepository(datastore.Get())
				if _, err := links.GetByID(cmd.Context(), id); errors.Is(err, repositories.ErrNotFound) {
					return errors.New(messages.CliModerateMissing(say, messages.CliModerateMissingParams{Slug: args[0]}))
				} else if err != nil {
					return err
				}
				return links.SetModeration(cmd.Context(), id, action, note)
			},
		}
		if action != models.ModerationNone {
			c.Flags().StringVar(&note, "note", "", messages.CliModerateNoteFlag(say))
		}
		return c
	}
	list := &cobra.Command{
		Use:   "list",
		Short: messages.CliModerateList(say),
		Args:  usage(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := openStore(cmd.Context()); err != nil {
				return err
			}
			items, err := repositories.NewLinkRepository(datastore.Get()).ListModeration(cmd.Context())
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, m := range items {
				rows = append(rows, []string{slug.MustEncode(m.LinkID), m.FullName, string(m.Action), m.Note})
			}
			printTable(stdout(cmd), []string{messages.CliColumnSlug(say), messages.CliColumnRepository(say),
				messages.CliColumnAction(say), messages.CliColumnNote(say)}, rows)
			return nil
		},
	}
	moderate.AddCommand(
		set(models.ModerationUnlist, "unlist", messages.CliModerateUnlist(say)),
		set(models.ModerationRemove, "remove", messages.CliModerateRemove(say)),
		set(models.ModerationNone, "clear", messages.CliModerateClear(say)),
		list,
	)
	return moderate
}
