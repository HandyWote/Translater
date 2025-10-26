<script lang="ts" setup>
import {useSettingsForm} from './useSettingsForm';

const form = useSettingsForm();
</script>

<template>
	<div class="settings-behavior">
		<label class="settings-toggle">
			<input v-model="form.autoCopyResult" type="checkbox" />
			<div>
				<strong>自动复制译文</strong>
				<span>翻译完成后写入剪贴板，便于直接粘贴到工作文档。</span>
			</div>
		</label>
		<label class="settings-toggle">
			<input v-model="form.keepWindowOnTop" type="checkbox" />
			<div>
				<strong>窗口保持置顶</strong>
				<span>让结果面板始终可见，提升校对效率。</span>
			</div>
		</label>
		<label class="settings-toggle">
			<input v-model="form.showToastOnComplete" type="checkbox" />
			<div>
				<strong>显示完成提示</strong>
				<span>屏幕底部弹出完成提示，及时知晓处理状态。</span>
			</div>
		</label>
	</div>
</template>

<style scoped>
.settings-behavior {
	display: flex;
	flex-direction: column;
	gap: 0.8rem;
}

.settings-toggle {
	display: flex;
	gap: 0.75rem;
	align-items: flex-start;
	padding: 0.7rem 0.85rem;
	border-radius: 12px;
	background: var(--surface-base);
	border: 1px solid var(--border-subtle);
	cursor: pointer;
	transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.settings-toggle input {
	margin-top: 0.3rem;
}

.settings-toggle strong {
	font-size: 0.92rem;
	font-weight: 600;
}

.settings-toggle span {
	display: block;
	margin-top: 0.2rem;
	color: var(--color-text-tertiary);
	font-size: 0.78rem;
	line-height: 1.35;
}
</style>
