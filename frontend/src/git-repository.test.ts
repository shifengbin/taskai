import {describe, expect, it} from 'vitest'

import {gitRepositoryPath} from './git-repository'

describe('gitRepositoryPath', () => {
	it('从 SSH、HTTPS 和 SCP 地址提取完整项目路径', () => {
		expect(gitRepositoryPath('git@gitlab.example.com:team/platform/api.git')).toBe('team/platform/api')
		expect(gitRepositoryPath('ssh://git@gitlab.example.com/team/platform/api.git')).toBe('team/platform/api')
		expect(gitRepositoryPath('https://gitlab.example.com/team/platform/api.git')).toBe('team/platform/api')
	})

	it('无法确认完整路径时返回空值', () => {
		expect(gitRepositoryPath('')).toBe('')
		expect(gitRepositoryPath('team/api')).toBe('')
		expect(gitRepositoryPath('git@gitlab.example.com:api.git')).toBe('')
		expect(gitRepositoryPath('https://gitlab.example.com/api.git')).toBe('')
	})
})
