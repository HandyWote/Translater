import {ref} from 'vue';

const DEFAULT_RESET_DELAY = 1600;

/**
 * 提供简单的剪贴板写入与提示状态管理。
 * 主要用于翻译结果等需要复制反馈的场景。
 */
export function useClipboard(resetDelay = DEFAULT_RESET_DELAY) {
	const copied = ref(false);
	const copying = ref(false);

	async function copy(text: string) {
		const content = text?.trim();
		if (!content || copying.value) {
			return false;
		}
		copying.value = true;
		try {
			await navigator.clipboard.writeText(content);
			copied.value = true;
			window.setTimeout(() => {
				copied.value = false;
			}, resetDelay);
			return true;
		} catch (error) {
			console.warn('复制失败', error);
			return false;
		} finally {
			copying.value = false;
		}
	}

	return {
		copy,
		copied,
		copying,
	};
}

export type ClipboardComposable = ReturnType<typeof useClipboard>;
