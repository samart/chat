# Protocol surface diff 0.3.300 -> 0.3.325

## sdkMessages (7)
- **changed-shape** `result:error_during_execution` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:error_max_budget_usd` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:error_max_structured_output_retries` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:error_max_turns` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:success` [additive, stable]: +credential_expires_at
- **changed-shape** `system:init` [neutral, stable]: ~plugin_errors
- **added** `system:session_writes_lost` [additive, stable] (SDKSessionWritesLostMessage: error_code, lost, session_id, subtype, transcript_session_ids, type, uuid)

## stdoutWire (7)
- **changed-shape** `result:error_during_execution` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:error_max_budget_usd` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:error_max_structured_output_retries` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:error_max_turns` [additive, stable]: +credential_expires_at; +invalid_config_file
- **changed-shape** `result:success` [additive, stable]: +credential_expires_at
- **changed-shape** `system:init` [neutral, stable]: ~plugin_errors
- **added** `system:session_writes_lost` [additive, stable] (SDKSessionWritesLostMessage: error_code, lost, session_id, subtype, transcript_session_ids, type, uuid)

## controlRequests (5)
- **changed-shape** `initialize` [neutral, stable]: ~sdkMcpServerConfigs
- **changed-shape** `mcp_call` [additive, stable]: +check_permissions; +result_not_for_model
- **changed-shape** `mcp_status` [additive, stable]: +liveReply
- **changed-shape** `reload_plugins` [additive, stable]: +add_plugins; +refuse_if_busy
- **added** `set_has_subscribers` [additive, stable] (SDKControlSetHasSubscribersRequest: has_subscribers, subtype)

## controlResponses (4)
- **changed-shape** `SDKControlGetContextUsageResponse` [neutral, stable]: ~agents; ~categories; +mcpInstructions; ~skills
- **changed-shape** `SDKControlGetUsageResponse` [neutral, stable]: ~behaviors
- **changed-shape** `SDKControlInitializeResponse` [additive, stable]: +hook_on_failure_supported
- **changed-shape** `SDKControlReadFileResponse` [additive, stable]: +lossy

## tools (13)
- **changed-shape** `AgentOutput` [additive, stable]
- **changed-shape** `ArtifactInput` [additive, stable]: +page_token
- **changed-shape** `ArtifactOutput` [additive, stable]
- **changed-shape** `AskUserQuestionOutput` [neutral, stable]: ~annotations
- **changed-shape** `BashOutput` [additive, stable]: +backgroundWakesOwner
- **changed-shape** `CronDeleteOutput` [additive, stable]: +endedLoopWakeup
- **changed-shape** `FileReadOutput` [additive, stable]
- **changed-shape** `GlobOutput` [additive, stable]: +timedOut
- **changed-shape** `GrepOutput` [additive, stable]: +timedOut
- **changed-shape** `ProjectsOutput` [additive, stable]
- **changed-shape** `ProposeSkillsInput` [neutral, stable]: ~proposals
- **changed-shape** `PushNotificationInput` [neutral, stable]: ~status
- **changed-shape** `TaskCreateOutput` [neutral, stable]: ~task

## queryMethods (1)
- **changed-shape** `reloadPlugins` [neutral, stable]: type: `…ptions?: { holdOnCacheImpact?: boolean; …` -> `…ptions?: { holdOnCacheImpact?: boolean; addPlugins?: SdkPluginConfig[]; refuseIfBusy?: boolean; …`

## settings (15)
- **changed-shape** `additionalMarketplaces` [neutral, stable]: type: `…ng; version?: string; strict?: boolean; headers?: { [k: string]: string; }; headersHelper?: string; }[]; owner?: { name: string; email?: string; url?: string; }…` -> `…ng; version?: string; strict?: boolean; engines?: { [k: string]: unknown; }; headers?: { [k: string]: string; }; headersHelper?: string; }[]; owner?: { name: string; email?: string; url?: string; }; }…`
- **changed-shape** `allowedMarketplaces` [neutral, stable]: type: `…ng; version?: string; strict?: boolean; …` -> `…ng; version?: string; strict?: boolean; engines?: { [k: string]: unknown; }; …`
- **added** `attributionProject` [additive, stable]: `string`
- **added** `attributionProjectPolicy` [additive, stable]: `{ default?: string; }`
- **added** `autoMode` [additive, stable]: `{ allow?: string[]; soft_deny?: string[]; hard_deny?: string[]; environment?: string[]; classifyAllShell?: boolean; }`
- **changed-shape** `blockedMarketplaces` [neutral, stable]: type: `…ng; version?: string; strict?: boolean; …` -> `…ng; version?: string; strict?: boolean; engines?: { [k: string]: unknown; }; …`
- **added** `disableReplyCheck` [additive, stable]: `boolean`
- **added** `disableWindowsBashCommandSplit` [additive, stable]: `boolean`
- **changed-shape** `extraKnownMarketplaces` [neutral, stable]: type: `…ng; version?: string; strict?: boolean; headers?: { [k: string]: string; }; headersHelper?: string; }[]; owner?: { name: string; email?: string; url?: string; }…` -> `…ng; version?: string; strict?: boolean; engines?: { [k: string]: unknown; }; headers?: { [k: string]: string; }; headersHelper?: string; }[]; owner?: { name: string; email?: string; url?: string; }; }…`
- **changed-shape** `permissions` [neutral, stable]: type: `…adsOutsideWorkingDirectories?: boolean; …` -> `…adsOutsideWorkingDirectories?: boolean; blockWritesOutsideWorkingDirectories?: boolean; blockReadsExemptPaths?: string[]; workingDirectoriesAsGiven?: string[]; disableAutoMode?: 'disable'; denyInstead…`
- **changed-shape** `sandbox` [neutral, stable]: type: `…}; enableWeakerNestedSandbox?: boolean; …` -> `…}; enableWeakerNestedSandbox?: boolean; allowNestedUserNamespaces?: boolean; …`
- **added** `skipAutoPermissionPrompt` [additive, stable]: `boolean`
- **added** `sshHostCommands` [additive, stable]: `unknown[]`
- **changed-shape** `strictKnownMarketplaces` [neutral, stable]: type: `…ng; version?: string; strict?: boolean; …` -> `…ng; version?: string; strict?: boolean; engines?: { [k: string]: unknown; }; …`
- **added** `useAutoModeDuringPlan` [additive, stable]: `boolean`

## exports (26)
- **added** `BUILTIN_TOOL_NAMES` [additive, stable]
- **added** `DeniedModelsMatch` [additive, stable]
- **changed-shape** `HookCallbackMatcher` [neutral, stable]
- **added** `LEGACY_TOOL_NAME_ALIASES` [additive, stable]
- **changed-shape** `McpSdkServerConfig` [neutral, stable]
- **changed-shape** `ModelUsage` [neutral, stable]
- **changed-shape** `Query` [neutral, stable]
- **changed-shape** `ResolvedSettings` [neutral, stable]
- **changed-shape** `SDKControlGetContextUsageResponse` [neutral, stable]
- **changed-shape** `SDKControlGetUsageResponse` [neutral, stable]
- **changed-shape** `SDKControlInitializeResponse` [neutral, stable]
- **changed-shape** `SDKControlPermissionRulesState` [neutral, stable]
- **changed-shape** `SDKControlReadFileResponse` [neutral, stable]
- **changed-shape** `SDKMessage` [neutral, stable]
- **changed-shape** `SDKPermissionRuleEntry` [neutral, stable]
- **changed-shape** `SDKResultError` [neutral, stable]
- **changed-shape** `SDKResultSuccess` [neutral, stable]
- **changed-shape** `SDKSessionInfo` [neutral, stable]
- **added** `SDKSessionWritesLostMessage` [additive, stable]
- **changed-shape** `SDKStartupFailureReason` [neutral, stable]
- **changed-shape** `SDKSystemMessage` [neutral, stable]
- **changed-shape** `Settings` [neutral, stable]
- **changed-shape** `USAGE_TRANSITION_PREFIXES` [neutral, stable]
- **added** `archiveSession` [additive, stable]
- **added** `matchDeniedModels` [additive, stable]
- **added** `unarchiveSession` [additive, stable]

## stringEnums (3)
- **added** `BUILTIN_TOOL_NAMES` [additive, stable]
- **changed-shape** `SDKStartupFailureReason` [additive, stable]: +['config_file_invalid', 'initialize_not_first', 'managed_settings_not_delivered', 'sandbox_required_unavailable'] -[]
- **changed-shape** `USAGE_TRANSITION_PREFIXES` [additive, stable]: +['Now using credits'] -[]

