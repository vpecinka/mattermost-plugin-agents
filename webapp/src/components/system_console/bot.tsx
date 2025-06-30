// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import styled from 'styled-components';
import {FormattedMessage, useIntl} from 'react-intl';

import {TrashCanOutlineIcon, ChevronDownIcon, AlertOutlineIcon, ChevronUpIcon, PlusIcon, CloseIcon} from '@mattermost/compass-icons/components';

import IconAI from '../assets/icon_ai';
import {DangerPill, Pill} from '../pill';

import {ButtonIcon} from '../assets/buttons';

import {BooleanItem, ItemList, SelectionItem, SelectionItemOption, TextItem, ItemLabel, HelpText} from './item';
import AvatarItem from './avatar';
import {ChannelAccessLevelItem, UserAccessLevelItem} from './llm_access';

export type LLMService = {
    type: string
    apiURL: string
    apiKey: string
    orgId: string
    defaultModel: string
    tokenLimit: number
    streamingTimeoutSeconds: number
    sendUserId: boolean
    outputTokenLimit: number
    customHeaders: {[key: string]: string}
}

export enum ChannelAccessLevel {
    All = 0,
    Allow,
    Block,
    None,
}

export enum UserAccessLevel {
    All = 0,
    Allow,
    Block,
    None,
}

export type LLMBotConfig = {
    id: string
    name: string
    displayName: string
    service: LLMService
    customInstructions: string
    enableVision: boolean
    disableTools: boolean
    channelAccessLevel: ChannelAccessLevel
    channelIDs: string[]
    userAccessLevel: UserAccessLevel
    userIDs: string[]
    teamIDs: string[]
}

type Props = {
    bot: LLMBotConfig
    onChange: (bot: LLMBotConfig) => void
    onDelete: () => void
    changedAvatar: (image: File) => void
}

const mapServiceTypeToDisplayName = new Map<string, string>([
    ['openai', 'OpenAI'],
    ['openaicompatible', 'OpenAI Compatible'],
    ['azure', 'Azure'],
    ['anthropic', 'Anthropic'],
]);

function serviceTypeToDisplayName(serviceType: string): string {
    return mapServiceTypeToDisplayName.get(serviceType) || serviceType;
}

const Bot = (props: Props) => {
    const [open, setOpen] = useState(false);
    const intl = useIntl();
    const missingInfo = props.bot.name === '' ||
		props.bot.displayName === '' ||
		props.bot.service.type === '' ||
		(props.bot.service.type !== 'openaicompatible' && props.bot.service.type !== 'azure' && props.bot.service.apiKey === '') ||
		((props.bot.service.type === 'openaicompatible' || props.bot.service.type === 'azure') && props.bot.service.apiURL === '');

    const invalidUsername = props.bot.name !== '' && (!(/^[a-z0-9.\-_]+$/).test(props.bot.name) || !(/[a-z]/).test(props.bot.name.charAt(0)));
    const invalidMaxTokens = props.bot.service.type === 'anthropic' && props.bot.service?.outputTokenLimit === 0;
    return (
        <BotContainer>
            <HeaderContainer onClick={() => setOpen((o) => !o)}>
                <IconAI/>
                <Title>
                    <NameText>
                        {props.bot.displayName}
                    </NameText>
                    <VerticalDivider/>
                    <ServiceTypeText>
                        {serviceTypeToDisplayName(props.bot.service.type)}
                    </ServiceTypeText>
                </Title>
                <Spacer/>
                {missingInfo && (
                    <DangerPill>
                        <AlertOutlineIcon/>
                        <FormattedMessage defaultMessage='Missing information'/>
                    </DangerPill>
                )}
                {invalidUsername && (
                    <DangerPill>
                        <AlertOutlineIcon/>
                        <FormattedMessage defaultMessage='Invalid Username'/>
                    </DangerPill>
                )}
                {invalidMaxTokens && (
                    <DangerPill>
                        <AlertOutlineIcon/>
                        <FormattedMessage defaultMessage='Output token limit must be greater than 0'/>
                    </DangerPill>
                )}

                <ButtonIcon
                    onClick={props.onDelete}
                >
                    <TrashIcon/>
                </ButtonIcon>
                {open ? <ChevronUpIcon/> : <ChevronDownIcon/>}
            </HeaderContainer>
            {open && (
                <ItemListContainer>
                    <ItemList>
                        <TextItem
                            label={intl.formatMessage({defaultMessage: 'Display name'})}
                            value={props.bot.displayName}
                            onChange={(e) => props.onChange({...props.bot, displayName: e.target.value})}
                        />
                        <TextItem
                            label={intl.formatMessage({defaultMessage: 'Bot Username'})}
                            helptext={intl.formatMessage({defaultMessage: 'Team members can mention this bot with this username'})}
                            maxLength={22}
                            value={props.bot.name}
                            onChange={(e) => props.onChange({...props.bot, name: e.target.value})}
                        />
                        <AvatarItem
                            botusername={props.bot.name}
                            changedAvatar={props.changedAvatar}
                        />
                        <SelectionItem
                            label={intl.formatMessage({defaultMessage: 'Service'})}
                            value={props.bot.service.type}
                            onChange={(e) => props.onChange({...props.bot, service: {...props.bot.service, type: e.target.value}})}
                        >
                            <SelectionItemOption value='openai'>{'OpenAI'}</SelectionItemOption>
                            <SelectionItemOption value='openaicompatible'>{'OpenAI Compatible'}</SelectionItemOption>
                            <SelectionItemOption value='azure'>{'Azure'}</SelectionItemOption>
                            <SelectionItemOption value='anthropic'>{'Anthropic'}</SelectionItemOption>
                        </SelectionItem>
                        <ServiceItem
                            service={props.bot.service}
                            onChange={(service) => props.onChange({...props.bot, service})}
                        />
                        <TextItem
                            label={intl.formatMessage({defaultMessage: 'Custom instructions'})}
                            placeholder={intl.formatMessage({defaultMessage: 'How would you like the AI to respond?'})}
                            multiline={true}
                            value={props.bot.customInstructions}
                            onChange={(e) => props.onChange({...props.bot, customInstructions: e.target.value})}
                        />
                        {(props.bot.service.type === 'openai' || props.bot.service.type === 'openaicompatible' || props.bot.service.type === 'azure' || props.bot.service.type === 'anthropic') && (
                            <>
                                <BooleanItem
                                    label={
                                        <Horizontal>
                                            <FormattedMessage defaultMessage='Enable Vision'/>
                                            <Pill><FormattedMessage defaultMessage='BETA'/></Pill>
                                        </Horizontal>
                                    }
                                    value={props.bot.enableVision}
                                    onChange={(to: boolean) => props.onChange({...props.bot, enableVision: to})}
                                    helpText={intl.formatMessage({defaultMessage: 'Enable Vision to allow the bot to process images. Requires a compatible model.'})}
                                />
                                <BooleanItem
                                    label={
                                        <FormattedMessage defaultMessage='Enable Tools'/>
                                    }
                                    value={!props.bot.disableTools}
                                    onChange={(to: boolean) => props.onChange({...props.bot, disableTools: !to})}
                                    helpText={intl.formatMessage({defaultMessage: 'By default some tool use is enabled to allow for features such as integrations with JIRA. Disabling this allows use of models that do not support or are not very good at tool use. Some features will not work without tools.'})}
                                />
                            </>
                        )}
                        <ChannelAccessLevelItem
                            label={intl.formatMessage({defaultMessage: 'Channel access'})}
                            level={props.bot.channelAccessLevel ?? ChannelAccessLevel.All}
                            onChangeLevel={(to: ChannelAccessLevel) => props.onChange({...props.bot, channelAccessLevel: to})}
                            channelIDs={props.bot.channelIDs ?? []}
                            onChangeChannelIDs={(channels: string[]) => props.onChange({...props.bot, channelIDs: channels})}
                        />
                        <UserAccessLevelItem
                            label={intl.formatMessage({defaultMessage: 'User access'})}
                            level={props.bot.userAccessLevel ?? ChannelAccessLevel.All}
                            onChangeLevel={(to: UserAccessLevel) => props.onChange({...props.bot, userAccessLevel: to})}
                            userIDs={props.bot.userIDs ?? []}
                            teamIDs={props.bot.teamIDs ?? []}
                            onChangeIDs={(userIds: string[], teamIds: string[]) => props.onChange({...props.bot, userIDs: userIds, teamIDs: teamIds})}
                        />
                        <CustomHeadersItem
                            customHeaders={props.bot.service.customHeaders}
                            onChange={(customHeaders) => props.onChange({...props.bot, service: {...props.bot.service, customHeaders}})}
                        />

                    </ItemList>
                </ItemListContainer>
            )}
        </BotContainer>
    );
};

const Horizontal = styled.div`
	display: flex;
	flex-direction: row;
	align-items: center;
	gap: 8px;
`;

type ServiceItemProps = {
    service: LLMService
    onChange: (service: LLMService) => void
}

const ServiceItem = (props: ServiceItemProps) => {
    const type = props.service.type;
    const intl = useIntl();
    const isOpenAIType = type === 'openai' || type === 'openaicompatible' || type === 'azure';

    const getDefaultOutputTokenLimit = () => {
        switch (type) {
        case 'anthropic':
            return '8192';
        default:
            return '0';
        }
    };

    return (
        <>
            {(type === 'openaicompatible' || type === 'azure') && (
                <TextItem
                    label={intl.formatMessage({defaultMessage: 'API URL'})}
                    value={props.service.apiURL}
                    onChange={(e) => props.onChange({...props.service, apiURL: e.target.value})}
                />
            )}
            <TextItem
                label={intl.formatMessage({defaultMessage: 'API Key'})}
                type='password'
                value={props.service.apiKey}
                onChange={(e) => props.onChange({...props.service, apiKey: e.target.value})}
            />
            {isOpenAIType && (
                <>
                    <TextItem
                        label={intl.formatMessage({defaultMessage: 'Organization ID'})}
                        value={props.service.orgId}
                        onChange={(e) => props.onChange({...props.service, orgId: e.target.value})}
                    />
                    <BooleanItem
                        label={intl.formatMessage({defaultMessage: 'Send User ID'})}
                        value={props.service.sendUserId}
                        onChange={(to: boolean) => props.onChange({...props.service, sendUserId: to})}
                        helpText={intl.formatMessage({defaultMessage: 'Sends the Mattermost user ID to the upstream LLM.'})}
                    />
                </>
            )}
            <TextItem
                label={intl.formatMessage({defaultMessage: 'Default model'})}
                value={props.service.defaultModel}
                onChange={(e) => props.onChange({...props.service, defaultModel: e.target.value})}
            />
            <TextItem
                label={intl.formatMessage({defaultMessage: 'Input token limit'})}
                type='number'
                value={props.service.tokenLimit.toString()}
                onChange={(e) => {
                    const value = parseInt(e.target.value, 10);
                    const tokenLimit = isNaN(value) ? 0 : value;
                    props.onChange({...props.service, tokenLimit});
                }}
            />
            <TextItem
                label={intl.formatMessage({defaultMessage: 'Output token limit'})}
                type='number'
                value={props.service.outputTokenLimit?.toString() || getDefaultOutputTokenLimit()}
                onChange={(e) => {
                    const value = parseInt(e.target.value, 10);
                    const outputTokenLimit = isNaN(value) ? 0 : value;
                    props.onChange({...props.service, outputTokenLimit});
                }}
            />
            {isOpenAIType && (
                <TextItem
                    label={intl.formatMessage({defaultMessage: 'Streaming Timeout Seconds'})}
                    type='number'
                    value={props.service.streamingTimeoutSeconds?.toString() || '0'}
                    onChange={(e) => {
                        const value = parseInt(e.target.value, 10);
                        const streamingTimeoutSeconds = isNaN(value) ? 0 : value;
                        props.onChange({...props.service, streamingTimeoutSeconds});
                    }}
                />
            )}
        </>
    );
};

const ItemListContainer = styled.div`
	padding: 24px 20px;
	padding-right: 76px;
`;

const Title = styled.div`
	display: flex;
	flex-direction: row;
	align-items: center;
	gap: 8px;
`;

const NameText = styled.div`
	font-size: 14px;
	font-weight: 600;
`;

const ServiceTypeText = styled.div`
	font-size: 14px;
	font-weight: 400;
	color: rgba(var(--center-channel-color-rgb), 0.72);
`;

const Spacer = styled.div`
	flex-grow: 1;
`;

const TrashIcon = styled(TrashCanOutlineIcon)`
	width: 16px;
	height: 16px;
	color: #D24B4E;
`;

const VerticalDivider = styled.div`
	width: 1px;
	border-left: 1px solid rgba(var(--center-channel-color-rgb), 0.16);
	height: 24px;
`;

const BotContainer = styled.div`
	display: flex;
	flex-direction: column;

	border-radius: 4px;
	border: 1px solid rgba(var(--center-channel-color-rgb), 0.12);

	&:hover {
		box-shadow: 0px 2px 3px 0px rgba(0, 0, 0, 0.08);
	}
`;

const HeaderContainer = styled.div`
	display: flex;
	flex-direction: row;
	justify-content: space-between;
	align-items: center;
	gap: 16px;
	padding: 12px 16px 12px 20px;
	border-bottom: 1px solid rgba(var(--center-channel-color-rgb), 0.12);
	cursor: pointer;
`;

const CustomHeadersContainer = styled.div`
    display: flex;
    flex-direction: column;
    gap: 8px;
`;

const HeaderRow = styled.div`
    display: flex;
    gap: 8px;
    align-items: center;
`;

const HeaderInput = styled.input`
    flex: 1;
    padding: 8px 12px;
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.16);
    border-radius: 4px;
    background: rgba(var(--center-channel-bg-rgb), 1);
    color: rgba(var(--center-channel-color-rgb), 1);
    font-size: 14px;
    
    &:focus {
        border-color: var(--button-bg);
        outline: none;
        box-shadow: 0 0 0 2px rgba(var(--button-bg-rgb), 0.2);
    }
    
    &::placeholder {
        color: rgba(var(--center-channel-color-rgb), 0.5);
    }
`;

const AddButton = styled.button`
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 8px 12px;
    border: 1px solid var(--button-bg);
    border-radius: 4px;
    background: transparent;
    color: var(--button-bg);
    font-size: 14px;
    cursor: pointer;
    
    &:hover {
        background: rgba(var(--button-bg-rgb), 0.08);
    }
`;

const RemoveButton = styled.button`
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.16);
    border-radius: 4px;
    background: transparent;
    color: rgba(var(--center-channel-color-rgb), 0.7);
    cursor: pointer;
    
    &:hover {
        background: rgba(var(--error-text-color-rgb), 0.08);
        border-color: var(--error-text-color);
        color: var(--error-text-color);
    }
`;

type CustomHeadersItemProps = {
    customHeaders: {[key: string]: string}
    onChange: (headers: {[key: string]: string}) => void
}

const CustomHeadersItem = (props: CustomHeadersItemProps) => {
    const intl = useIntl();
    const headers = Object.entries(props.customHeaders || {});
    
    // Generate stable keys for React to prevent focus loss
    const headersWithStableKeys = headers.map((header, index) => ({
        key: `header-${index}`,
        headerKey: header[0],
        value: header[1]
    }));
    
    const addHeader = () => {
        const newHeaders = {...props.customHeaders};
        // Find a unique placeholder name
        let counter = 1;
        while (newHeaders[`X-Custom-Header-${counter}`]) {
            counter++;
        }
        newHeaders[`X-Custom-Header-${counter}`] = '';
        props.onChange(newHeaders);
    };
    
    const updateHeaderKey = (oldKey: string, newKey: string) => {
        if (oldKey === newKey) return;
        
        const newHeaders = {...props.customHeaders};
        const value = newHeaders[oldKey];
        delete newHeaders[oldKey];
        if (newKey && !newHeaders[newKey]) {
            newHeaders[newKey] = value;
        }
        props.onChange(newHeaders);
    };
    
    const updateHeaderValue = (key: string, value: string) => {
        const newHeaders = {...props.customHeaders};
        newHeaders[key] = value;
        props.onChange(newHeaders);
    };
    
    const removeHeader = (key: string) => {
        const newHeaders = {...props.customHeaders};
        delete newHeaders[key];
        props.onChange(newHeaders);
    };
    
    return (
        <>
            <ItemLabel>
                {intl.formatMessage({defaultMessage: 'Custom Headers'})}
            </ItemLabel>
            <CustomHeadersContainer>
                {headersWithStableKeys.map((item) => (
                    <HeaderRow key={item.key}>
                        <HeaderInput
                            placeholder="Header name (e.g., X-Organization)"
                            value={item.headerKey}
                            onChange={(e) => updateHeaderKey(item.headerKey, e.target.value)}
                        />
                        <HeaderInput
                            placeholder="Header value"
                            value={item.value}
                            onChange={(e) => updateHeaderValue(item.headerKey, e.target.value)}
                        />
                        <RemoveButton
                            type="button"
                            onClick={() => removeHeader(item.headerKey)}
                            title="Remove header"
                        >
                            <CloseIcon size={16} />
                        </RemoveButton>
                    </HeaderRow>
                ))}
                <AddButton
                    type="button"
                    onClick={addHeader}
                >
                    <PlusIcon size={16} />
                    {intl.formatMessage({defaultMessage: 'Add Header'})}
                </AddButton>
                <HelpText>
                    {intl.formatMessage({defaultMessage: 'Custom headers will be sent with every API request to the LLM provider. Use this for authentication, tracking, or routing purposes.'})}
                </HelpText>
            </CustomHeadersContainer>
        </>
    );
};

export default Bot;
