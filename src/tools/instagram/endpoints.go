package instagram

import "github.com/thinktwiceco/agent-forge/src/tools/api"

const baseURL = "https://graph.instagram.com/v18.0"

var endpoints = []api.Endpoint{
	{
		Name:        "get_profile",
		URL:         baseURL + "/me",
		Method:      "GET",
		Description: "Get the authenticated user's Instagram profile fields",
		QueryParams: "fields: string - Comma-separated list of fields to return. Available: id,username,name,biography,followers_count,follows_count,media_count,profile_picture_url,website (default: id,username,name,biography,followers_count,follows_count,media_count,profile_picture_url,website)",
	},
	{
		Name:        "list_media",
		URL:         baseURL + "/me/media",
		Method:      "GET",
		Description: "List media posts published on the authenticated user's account",
		QueryParams: "fields: string - Comma-separated fields to return. Available: id,caption,media_type,media_url,thumbnail_url,timestamp,permalink,like_count,comments_count (default: id,caption,media_type,media_url,timestamp,permalink,like_count,comments_count)\nlimit: integer - Maximum number of posts to return (default: 20, max: 100)\nafter: string - Cursor for forward pagination",
	},
	{
		Name:          "get_media",
		URL:           baseURL + "/{media_id}",
		Method:        "GET",
		Description:   "Get details for a single media post by its ID",
		URLParameters: "media_id: string - The ID of the media post to retrieve",
		QueryParams:   "fields: string - Comma-separated fields to return. Available: id,caption,media_type,media_url,thumbnail_url,timestamp,permalink,like_count,comments_count,media_product_type",
	},
	{
		Name:   "create_media_container",
		URL:    baseURL + "/me/media",
		Method: "POST",
		Description: "Create a media container (step 1 of publishing). Returns a container ID.\n" +
			"For single image: provide image_url. For video/reel: provide video_url and set media_type to VIDEO or REELS.\n" +
			"For carousel item: set is_carousel_item to true. For story: set media_type to STORIES.",
		Payload: "image_url: string - Publicly accessible URL of the image to upload (JPEG only). Required for image posts.\n" +
			"video_url: string - Publicly accessible URL of the video to upload. Required for VIDEO, REELS, or STORIES media_type.\n" +
			"media_type: string - Type of media. Values: IMAGE (default), VIDEO, REELS, STORIES, CAROUSEL.\n" +
			"caption: string - Caption for the post. @ symbols are ignored unless admin-equivalent.\n" +
			"is_carousel_item: boolean - Set to true if this container is an item within a carousel.\n" +
			"children: string - Comma-separated container IDs for carousel items (required when media_type is CAROUSEL).\n" +
			"alt_text: string - Accessible alt text for image posts.",
	},
	{
		Name:        "publish_media",
		URL:         baseURL + "/me/media_publish",
		Method:      "POST",
		Description: "Publish a media container (step 2 of publishing). Provide the container ID returned by create_media_container.",
		Payload:     "creation_id: string - The container ID returned by create_media_container. Required.",
	},
	{
		Name:        "get_account_insights",
		URL:         baseURL + "/me/insights",
		Method:      "GET",
		Description: "Get aggregated interaction metrics for the authenticated account",
		QueryParams: "metric: string - Comma-separated metrics to retrieve. Values: impressions, reach, profile_views, accounts_engaged, total_interactions. Required.\n" +
			"period: string - Aggregation period. Values: day, week, month, lifetime (default: day).\n" +
			"since: string - Unix timestamp for the start of the range (optional).\n" +
			"until: string - Unix timestamp for the end of the range (optional).",
	},
	{
		Name:          "get_media_insights",
		URL:           baseURL + "/{media_id}/insights",
		Method:        "GET",
		Description:   "Get engagement metrics for a specific media post",
		URLParameters: "media_id: string - The ID of the media post",
		QueryParams:   "metric: string - Comma-separated metrics. Values: impressions, reach, engagement, saved, video_views, likes, comments, shares. Required.",
	},
	{
		Name:          "get_comments",
		URL:           baseURL + "/{media_id}/comments",
		Method:        "GET",
		Description:   "List comments on a media post",
		URLParameters: "media_id: string - The ID of the media post",
		QueryParams: "fields: string - Comma-separated fields to return. Available: id,text,timestamp,username (default: id,text,timestamp)\n" +
			"limit: integer - Maximum number of comments to return (default: 20)\n" +
			"after: string - Cursor for forward pagination",
	},
	{
		Name:          "reply_to_comment",
		URL:           baseURL + "/{comment_id}/replies",
		Method:        "POST",
		Description:   "Reply to a comment on a media post",
		URLParameters: "comment_id: string - The ID of the comment to reply to",
		Payload:       "message: string - The reply text. Required.",
	},
}
