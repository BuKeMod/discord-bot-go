package feature

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type embedContextKey struct{}

type ResponseHolder struct {
	Embed      *discordgo.MessageEmbed
	Embeds     []*discordgo.MessageEmbed
	Components []discordgo.MessageComponent
}

func ContextWithResponse(ctx context.Context) (context.Context, *ResponseHolder) {
	h := &ResponseHolder{}
	return context.WithValue(ctx, embedContextKey{}, h), h
}

func SetEmbed(ctx context.Context, embed *discordgo.MessageEmbed) {
	if h, ok := ctx.Value(embedContextKey{}).(*ResponseHolder); ok && h != nil {
		h.Embed = embed
	}
}

func SetEmbeds(ctx context.Context, embeds []*discordgo.MessageEmbed) {
	if h, ok := ctx.Value(embedContextKey{}).(*ResponseHolder); ok && h != nil {
		h.Embeds = embeds
	}
}

func SetComponents(ctx context.Context, components []discordgo.MessageComponent) {
	if h, ok := ctx.Value(embedContextKey{}).(*ResponseHolder); ok && h != nil {
		h.Components = components
	}
}
