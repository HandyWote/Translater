export const SECTION_STORAGE_KEY = 'settings-panel:sections';

export const sectionDefaults = {
	api: true,
	models: false,
	behavior: true,
	prompts: false,
	hotkey: true,
	theme: true,
} as const;

export type SectionKey = keyof typeof sectionDefaults;

export type CategoryKey = 'integration' | 'experience' | 'productivity' | 'appearance';

export interface SettingsCategory {
	key: CategoryKey;
	label: string;
	description: string;
	icon: string;
	sections: SectionKey[];
}

export const settingsCategories: SettingsCategory[] = [
	{key: 'integration', label: '服务能力', description: '统筹接口凭证与模型策略，确保端到端可用性。', icon: '🔌', sections: ['api', 'models']},
	{key: 'experience', label: '工作流体验', description: '调优翻译后的自动化动作与提示词，贴合团队流程。', icon: '⚙️', sections: ['behavior', 'prompts']},
	{key: 'productivity', label: '效率工具', description: '统一热键与交互方式，保持操作一致性。', icon: '⌨️', sections: ['hotkey']},
	{key: 'appearance', label: '界面主题', description: '设置主题与视觉偏好，营造舒适的使用体验。', icon: '🎨', sections: ['theme']},
];
