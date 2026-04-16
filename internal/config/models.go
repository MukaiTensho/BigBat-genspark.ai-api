package config

import "strings"

var DefaultModelList = []string{
	"gpt-5-pro",
	"gpt-5.1-low",
	"gpt-5.2",
	"gpt-5.2-pro",
	"o3-pro",
	"claude-sonnet-4-6",
	"claude-sonnet-4-5",
	"claude-opus-4-6",
	"claude-opus-4-5",
	"claude-4-5-haiku",
	"gemini-2.5-pro",
	"gemini-3-flash-preview",
	"gemini-3.1-pro-preview",
	"gemini-3-pro-preview",
	"grok-4-0709",

	"nano-banana-pro",
	"nano-banana-2",
	"fal-ai/bytedance/seedream/v5/lite",
	"fal-ai/flux-2",
	"fal-ai/flux-2-pro",
	"fal-ai/z-image/turbo",
	"fal-ai/gpt-image-1.5",
	"recraft-v3",
	"ideogram/V_3",
	"qwen-image",
	"fal-ai/recraft-clarity-upscale",
	"fal-bria-rmbg",
	"fal-ai/image-editing/text-removal",

	"gemini/veo3.1",
	"gemini/veo3.1/reference-to-video",
	"gemini/veo3.1/first-last-frame-to-video",
	"sora-2",
	"sora-2-pro",
	"gemini/veo3",
	"kling/v3",
	"kling/v2.6/standard/motion-control",
	"kling/o3/image-to-video",
	"kling/o3/reference-to-video",
	"fal-ai/bytedance/seedance/v1.5/pro",
	"xai/grok-imagine-video",
	"minimax/hailuo-2.3/standard",
	"official/pixverse/v5",
	"fal-ai/bytedance/seedance/v1/pro/fast",
	"fal-ai/sync-lipsync/v2",
	"wan/v2.6",
	"vidu/q3",
	"runway/gen4_turbo",
	"fal-ai/bytedance-upscaler/upscale/video",
}

var TextModelList = []string{
	"gpt-5-pro",
	"gpt-5.1-low",
	"gpt-5.2",
	"gpt-5.2-pro",
	"o3-pro",
	"claude-sonnet-4-6",
	"claude-sonnet-4-5",
	"claude-opus-4-6",
	"claude-opus-4-5",
	"claude-4-5-haiku",
	"gemini-2.5-pro",
	"gemini-3-flash-preview",
	"gemini-3.1-pro-preview",
	"gemini-3-pro-preview",
	"grok-4-0709",
}

var MixtureModelList = []string{
	"gpt-5.1-low",
	"claude-sonnet-4-5",
	"gemini-3-pro-preview",
}

var ImageModelList = []string{
	"nano-banana-pro",
	"nano-banana-2",
	"fal-ai/bytedance/seedream/v5/lite",
	"fal-ai/flux-2",
	"fal-ai/flux-2-pro",
	"fal-ai/z-image/turbo",
	"fal-ai/gpt-image-1.5",
	"recraft-v3",
	"ideogram/V_3",
	"qwen-image",
	"fal-ai/recraft-clarity-upscale",
	"fal-bria-rmbg",
	"fal-ai/image-editing/text-removal",
}

var VideoModelList = []string{
	"gemini/veo3.1",
	"gemini/veo3.1/reference-to-video",
	"gemini/veo3.1/first-last-frame-to-video",
	"sora-2",
	"sora-2-pro",
	"gemini/veo3",
	"kling/v3",
	"kling/v2.6/standard/motion-control",
	"kling/o3/image-to-video",
	"kling/o3/reference-to-video",
	"fal-ai/bytedance/seedance/v1.5/pro",
	"xai/grok-imagine-video",
	"minimax/hailuo-2.3/standard",
	"official/pixverse/v5",
	"fal-ai/bytedance/seedance/v1/pro/fast",
	"fal-ai/sync-lipsync/v2",
	"wan/v2.6",
	"vidu/q3",
	"runway/gen4_turbo",
	"fal-ai/bytedance-upscaler/upscale/video",
}

var ModelAliasMap = map[string]string{
	"opus4.6":         "claude-opus-4-6",
	"opus-4.6":        "claude-opus-4-6",
	"claude-opus-4.6": "claude-opus-4-6",
	"sonnet4.6":       "claude-sonnet-4-6",
	"sonnet-4.6":      "claude-sonnet-4-6",
	"gpt5pro":         "gpt-5-pro",
}

func ResolveModelAlias(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return model
	}
	if target, ok := ModelAliasMap[strings.ToLower(model)]; ok {
		return target
	}
	return model
}

func IsTextModel(model string) bool {
	return contains(TextModelList, model)
}

func IsImageModel(model string) bool {
	return contains(ImageModelList, model)
}

func IsVideoModel(model string) bool {
	return contains(VideoModelList, model)
}

func contains(list []string, needle string) bool {
	for _, item := range list {
		if item == needle {
			return true
		}
	}
	return false
}
