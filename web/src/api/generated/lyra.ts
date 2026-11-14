import {
  useMutation,
  useQuery
} from '@tanstack/react-query';
import type {
  DataTag,
  DefinedInitialDataOptions,
  DefinedUseQueryResult,
  MutationFunction,
  QueryClient,
  QueryFunction,
  QueryKey,
  UndefinedInitialDataOptions,
  UseMutationOptions,
  UseMutationResult,
  UseQueryOptions,
  UseQueryResult
} from '@tanstack/react-query';

import type {
  AssetWrite,
  BlobResponse,
  CommentRequest,
  CommentWrite,
  Commit,
  CommitWorkspace,
  CreateIssueRequest,
  CreateKeyRequest,
  CreateLabelRequest,
  CreateOwnerRequest,
  CreatePullRequest,
  CreateReleaseRequest,
  CreateRepoRequest,
  CreateTokenRequest,
  DeleteReleaseParams,
  DeleteRuleParams,
  DeletedID,
  DeletedIDWithEvent,
  DeletedName,
  DeletedWithEvent,
  DiffFile,
  DispatchRequest,
  EditItemRequest,
  EditReleaseRequest,
  ErrorBody,
  Event,
  FileChange,
  GetJobLogsParams,
  GetRepositoryBlobParams,
  GetRepositoryRawFileParams,
  GetRepositoryTreeParams,
  Image,
  Item,
  ItemView,
  ItemWorkspace,
  ItemWrite,
  Key,
  Label,
  LabelWrite,
  ListCommitsParams,
  ListEventsParams,
  ListIssuesParams,
  ListPullsParams,
  ListRepositoryFilesParams,
  ListRunsParams,
  ListWorkflowsParams,
  MarkdownRequest,
  MeView,
  MergeRequest,
  NameRequest,
  OwnerGroup,
  OwnerWrite,
  PreviewReleaseNotesParams,
  Pull,
  PullView,
  PullWrite,
  ReadOnlyRequest,
  RegistryCollected,
  Release,
  ReleaseDeleted,
  ReleaseNotes,
  ReleaseWrite,
  RenderedMarkdown,
  RepoView,
  RerunRunParams,
  ReviewRequest,
  ReviewWrite,
  RevokedID,
  Rule,
  RuleWrite,
  Run,
  RunCancelled,
  RunWrite,
  RunnerCreated,
  RunnerToken,
  RunnerTokenCreated,
  RunnerView,
  Secret,
  SecretSet,
  SetRuleRequest,
  SigningKeyView,
  StatusView,
  StorageCheck,
  Tag,
  Token,
  TokenCreated,
  TreeResponse,
  UploadAssetParams,
  ValueRequest,
  Variable,
  VariableSet,
  WorkflowWorkspace
} from './models.ts';

import { orvalFetch } from '../transport.ts';
import type { ErrorType } from '../transport.ts';


type SecondParameter<T extends (...args: never) => unknown> = Parameters<T>[1];



const withQueryKey = <T extends object, K>(query: T, queryKey: K): T & { queryKey: K } => {
  const result = { queryKey } as T & { queryKey: K };
  for (const key of Object.keys(query)) {
    // The explicit queryKey always wins, matching the previous
    // `{ ...query, queryKey }` spread where it was set last.
    if (key === 'queryKey') continue;
    Object.defineProperty(result, key, {
      enumerable: true,
      configurable: true,
      get: () => (query as Record<string, unknown>)[key],
    });
  }
  return result;
};

export const getSetReadOnlyUrl = () => {




  return `/api/admin/readonly`
}

/**
 * Scope: admin, unrestricted. The mode lives in memory, so a restart turns it off.
 * @summary Turn read-only mode on or off
 */
export const setReadOnly = async (readOnlyRequest: ReadOnlyRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<StatusView> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<StatusView>(getSetReadOnlyUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(readOnlyRequest)
  }
);}





export const getSetReadOnlyMutationKey = () => ['setReadOnly'] as const;

export const getSetReadOnlyMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setReadOnly>>, TError,SetReadOnlyMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof setReadOnly>>, TError,SetReadOnlyMutationVariables, TContext> => {

const mutationKey = getSetReadOnlyMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof setReadOnly>>, SetReadOnlyMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  setReadOnly(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SetReadOnlyMutationResult = NonNullable<Awaited<ReturnType<typeof setReadOnly>>>
    export type SetReadOnlyMutationBody = ReadOnlyRequest
    export type SetReadOnlyMutationError = ErrorType<ErrorBody>
    export type SetReadOnlyMutationVariables = {data: ReadOnlyRequest}

    /**
 * Scope: admin, unrestricted. The mode lives in memory, so a restart turns it off.
 * @summary Turn read-only mode on or off
 */
export const useSetReadOnly = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setReadOnly>>, TError,SetReadOnlyMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof setReadOnly>>,
        TError,
        SetReadOnlyMutationVariables,
        TContext
      > => {
      return useMutation(getSetReadOnlyMutationOptions(options), queryClient);
    }

export const getCollectRegistryUrl = () => {




  return `/api/admin/registry-gc`
}

/**
 * Scope: admin, unrestricted. Lyra is read-only while it runs.
 * @summary Delete registry layers and manifests no tag refers to
 */
export const collectRegistry = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<RegistryCollected> => {

  return orvalFetch<RegistryCollected>(getCollectRegistryUrl(),
  {
    ...options,
    method: 'POST'


  }
);}





export const getCollectRegistryMutationKey = () => ['collectRegistry'] as const;

export const getCollectRegistryMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof collectRegistry>>, TError,void, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof collectRegistry>>, TError,void, TContext> => {

const mutationKey = getCollectRegistryMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof collectRegistry>>, void> = () => {


          return  collectRegistry(requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CollectRegistryMutationResult = NonNullable<Awaited<ReturnType<typeof collectRegistry>>>

    export type CollectRegistryMutationError = ErrorType<ErrorBody>


    /**
 * Scope: admin, unrestricted. Lyra is read-only while it runs.
 * @summary Delete registry layers and manifests no tag refers to
 */
export const useCollectRegistry = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof collectRegistry>>, TError,void, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof collectRegistry>>,
        TError,
        void,
        TContext
      > => {
      return useMutation(getCollectRegistryMutationOptions(options), queryClient);
    }

export const getGetStatusUrl = () => {




  return `/api/admin/status`
}

/**
 * Scope: admin, unrestricted.
 * @summary Read-only mode and the last backup
 */
export const getStatus = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<StatusView> => {

  return orvalFetch<StatusView>(getGetStatusUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetStatusQueryKey = () => {
    return [
    `/api/admin/status`
    ] as const;
    }


export const getGetStatusQueryOptions = <TData = Awaited<ReturnType<typeof getStatus>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetStatusQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getStatus>>> = ({ signal }) => getStatus({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetStatusQueryResult = NonNullable<Awaited<ReturnType<typeof getStatus>>>
export type GetStatusQueryError = ErrorType<ErrorBody>


/**
 * Scope: admin, unrestricted.
 * @summary Read-only mode and the last backup
 */
export function useGetStatus<TData = Awaited<ReturnType<typeof getStatus>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getStatus>>,
          TError,
          Awaited<ReturnType<typeof getStatus>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary Read-only mode and the last backup
 */
export function useGetStatus<TData = Awaited<ReturnType<typeof getStatus>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getStatus>>,
          TError,
          Awaited<ReturnType<typeof getStatus>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary Read-only mode and the last backup
 */
export function useGetStatus<TData = Awaited<ReturnType<typeof getStatus>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary Read-only mode and the last backup
 */
export function useGetStatus<TData = Awaited<ReturnType<typeof getStatus>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getStatus>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetStatusQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCheckStorageUrl = () => {




  return `/api/admin/storage-check`
}

/**
 * Scope: admin, unrestricted.
 * @summary Stored files the database refers to but storage lacks
 */
export const checkStorage = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<StorageCheck> => {

  return orvalFetch<StorageCheck>(getCheckStorageUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getCheckStorageQueryKey = () => {
    return [
    `/api/admin/storage-check`
    ] as const;
    }


export const getCheckStorageQueryOptions = <TData = Awaited<ReturnType<typeof checkStorage>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof checkStorage>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getCheckStorageQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof checkStorage>>> = ({ signal }) => checkStorage({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof checkStorage>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type CheckStorageQueryResult = NonNullable<Awaited<ReturnType<typeof checkStorage>>>
export type CheckStorageQueryError = ErrorType<ErrorBody>


/**
 * Scope: admin, unrestricted.
 * @summary Stored files the database refers to but storage lacks
 */
export function useCheckStorage<TData = Awaited<ReturnType<typeof checkStorage>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof checkStorage>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof checkStorage>>,
          TError,
          Awaited<ReturnType<typeof checkStorage>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary Stored files the database refers to but storage lacks
 */
export function useCheckStorage<TData = Awaited<ReturnType<typeof checkStorage>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof checkStorage>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof checkStorage>>,
          TError,
          Awaited<ReturnType<typeof checkStorage>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary Stored files the database refers to but storage lacks
 */
export function useCheckStorage<TData = Awaited<ReturnType<typeof checkStorage>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof checkStorage>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary Stored files the database refers to but storage lacks
 */
export function useCheckStorage<TData = Awaited<ReturnType<typeof checkStorage>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof checkStorage>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getCheckStorageQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetCodeStylesUrl = () => {




  return `/api/code.css`
}

/**
 * Scope: any.
 * @summary Read syntax highlighting styles for light and dark appearance
 */
export const getCodeStyles = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<string> => {

  return orvalFetch<string>(getGetCodeStylesUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetCodeStylesQueryKey = () => {
    return [
    `/api/code.css`
    ] as const;
    }


export const getGetCodeStylesQueryOptions = <TData = Awaited<ReturnType<typeof getCodeStyles>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCodeStyles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetCodeStylesQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getCodeStyles>>> = ({ signal }) => getCodeStyles({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getCodeStyles>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetCodeStylesQueryResult = NonNullable<Awaited<ReturnType<typeof getCodeStyles>>>
export type GetCodeStylesQueryError = ErrorType<ErrorBody>


/**
 * Scope: any.
 * @summary Read syntax highlighting styles for light and dark appearance
 */
export function useGetCodeStyles<TData = Awaited<ReturnType<typeof getCodeStyles>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCodeStyles>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getCodeStyles>>,
          TError,
          Awaited<ReturnType<typeof getCodeStyles>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary Read syntax highlighting styles for light and dark appearance
 */
export function useGetCodeStyles<TData = Awaited<ReturnType<typeof getCodeStyles>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCodeStyles>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getCodeStyles>>,
          TError,
          Awaited<ReturnType<typeof getCodeStyles>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary Read syntax highlighting styles for light and dark appearance
 */
export function useGetCodeStyles<TData = Awaited<ReturnType<typeof getCodeStyles>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCodeStyles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary Read syntax highlighting styles for light and dark appearance
 */
export function useGetCodeStyles<TData = Awaited<ReturnType<typeof getCodeStyles>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCodeStyles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetCodeStylesQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListEventsUrl = (params?: ListEventsParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/events?${stringifiedParams}` : `/api/events`
}

/**
 * Scope: read. Account events (tokens, keys) need an unrestricted token.
 * @summary List recent events
 */
export const listEvents = async (params?: ListEventsParams, options?: Parameters<typeof orvalFetch>[1]): Promise<Event[]> => {

  return orvalFetch<Event[]>(getListEventsUrl(params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListEventsQueryKey = (params?: ListEventsParams,) => {
    return [
    `/api/events`, ...(params ? [params] : [])
    ] as const;
    }


export const getListEventsQueryOptions = <TData = Awaited<ReturnType<typeof listEvents>>, TError = ErrorType<ErrorBody>>(params?: ListEventsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listEvents>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListEventsQueryKey(params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listEvents>>> = ({ signal }) => listEvents(params, { signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listEvents>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListEventsQueryResult = NonNullable<Awaited<ReturnType<typeof listEvents>>>
export type ListEventsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read. Account events (tokens, keys) need an unrestricted token.
 * @summary List recent events
 */
export function useListEvents<TData = Awaited<ReturnType<typeof listEvents>>, TError = ErrorType<ErrorBody>>(
 params: undefined |  ListEventsParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listEvents>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listEvents>>,
          TError,
          Awaited<ReturnType<typeof listEvents>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Account events (tokens, keys) need an unrestricted token.
 * @summary List recent events
 */
export function useListEvents<TData = Awaited<ReturnType<typeof listEvents>>, TError = ErrorType<ErrorBody>>(
 params?: ListEventsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listEvents>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listEvents>>,
          TError,
          Awaited<ReturnType<typeof listEvents>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Account events (tokens, keys) need an unrestricted token.
 * @summary List recent events
 */
export function useListEvents<TData = Awaited<ReturnType<typeof listEvents>>, TError = ErrorType<ErrorBody>>(
 params?: ListEventsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listEvents>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Account events (tokens, keys) need an unrestricted token.
 * @summary List recent events
 */
export function useListEvents<TData = Awaited<ReturnType<typeof listEvents>>, TError = ErrorType<ErrorBody>>(
 params?: ListEventsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listEvents>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListEventsQueryOptions(params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListKeysUrl = () => {




  return `/api/keys`
}

/**
 * Scope: read, unrestricted.
 * @summary List the caller's SSH keys
 */
export const listKeys = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<Key[]> => {

  return orvalFetch<Key[]>(getListKeysUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListKeysQueryKey = () => {
    return [
    `/api/keys`
    ] as const;
    }


export const getListKeysQueryOptions = <TData = Awaited<ReturnType<typeof listKeys>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listKeys>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListKeysQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listKeys>>> = ({ signal }) => listKeys({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listKeys>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListKeysQueryResult = NonNullable<Awaited<ReturnType<typeof listKeys>>>
export type ListKeysQueryError = ErrorType<ErrorBody>


/**
 * Scope: read, unrestricted.
 * @summary List the caller's SSH keys
 */
export function useListKeys<TData = Awaited<ReturnType<typeof listKeys>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listKeys>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listKeys>>,
          TError,
          Awaited<ReturnType<typeof listKeys>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read, unrestricted.
 * @summary List the caller's SSH keys
 */
export function useListKeys<TData = Awaited<ReturnType<typeof listKeys>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listKeys>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listKeys>>,
          TError,
          Awaited<ReturnType<typeof listKeys>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read, unrestricted.
 * @summary List the caller's SSH keys
 */
export function useListKeys<TData = Awaited<ReturnType<typeof listKeys>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listKeys>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read, unrestricted.
 * @summary List the caller's SSH keys
 */
export function useListKeys<TData = Awaited<ReturnType<typeof listKeys>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listKeys>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListKeysQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateKeyUrl = () => {




  return `/api/keys`
}

/**
 * Scope: admin, unrestricted. Signing keys verify commit and tag signatures.
 * @summary Register an SSH signing key
 */
export const createKey = async (createKeyRequest: CreateKeyRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<Key> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<Key>(getCreateKeyUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createKeyRequest)
  }
);}





export const getCreateKeyMutationKey = () => ['createKey'] as const;

export const getCreateKeyMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createKey>>, TError,CreateKeyMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createKey>>, TError,CreateKeyMutationVariables, TContext> => {

const mutationKey = getCreateKeyMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createKey>>, CreateKeyMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  createKey(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateKeyMutationResult = NonNullable<Awaited<ReturnType<typeof createKey>>>
    export type CreateKeyMutationBody = CreateKeyRequest
    export type CreateKeyMutationError = ErrorType<ErrorBody>
    export type CreateKeyMutationVariables = {data: CreateKeyRequest}

    /**
 * Scope: admin, unrestricted. Signing keys verify commit and tag signatures.
 * @summary Register an SSH signing key
 */
export const useCreateKey = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createKey>>, TError,CreateKeyMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createKey>>,
        TError,
        CreateKeyMutationVariables,
        TContext
      > => {
      return useMutation(getCreateKeyMutationOptions(options), queryClient);
    }

export const getDeleteKeyUrl = (id: number,) => {




  return `/api/keys/${encodeURIComponent(String(id))}`
}

/**
 * Scope: admin, unrestricted.
 * @summary Delete an SSH key
 */
export const deleteKey = async (id: number, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedID> => {

  return orvalFetch<DeletedID>(getDeleteKeyUrl(id),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteKeyMutationKey = () => ['deleteKey'] as const;

export const getDeleteKeyMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteKey>>, TError,DeleteKeyMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteKey>>, TError,DeleteKeyMutationVariables, TContext> => {

const mutationKey = getDeleteKeyMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteKey>>, DeleteKeyMutationVariables> = (props) => {
          const {id} = props ?? {};

          return  deleteKey(id,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteKeyMutationResult = NonNullable<Awaited<ReturnType<typeof deleteKey>>>

    export type DeleteKeyMutationError = ErrorType<ErrorBody>
    export type DeleteKeyMutationVariables = {id: number}

    /**
 * Scope: admin, unrestricted.
 * @summary Delete an SSH key
 */
export const useDeleteKey = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteKey>>, TError,DeleteKeyMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteKey>>,
        TError,
        DeleteKeyMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteKeyMutationOptions(options), queryClient);
    }

export const getRenderMarkdownUrl = () => {




  return `/api/markdown`
}

/**
 * Scope: any. This read-only operation remains available during maintenance.
 * @summary Preview sanitized Markdown
 */
export const renderMarkdown = async (markdownRequest: MarkdownRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<RenderedMarkdown> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<RenderedMarkdown>(getRenderMarkdownUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(markdownRequest)
  }
);}





export const getRenderMarkdownQueryKey = (markdownRequest?: MarkdownRequest,) => {
    return [
    'POST', `/api/markdown`, markdownRequest
    ] as const;
    }


export const getRenderMarkdownQueryOptions = <TData = Awaited<ReturnType<typeof renderMarkdown>>, TError = ErrorType<ErrorBody>>(markdownRequest: MarkdownRequest, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof renderMarkdown>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getRenderMarkdownQueryKey(markdownRequest);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof renderMarkdown>>> = ({ signal }) => renderMarkdown(markdownRequest, { signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof renderMarkdown>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type RenderMarkdownQueryResult = NonNullable<Awaited<ReturnType<typeof renderMarkdown>>>
export type RenderMarkdownQueryError = ErrorType<ErrorBody>


/**
 * Scope: any. This read-only operation remains available during maintenance.
 * @summary Preview sanitized Markdown
 */
export function useRenderMarkdown<TData = Awaited<ReturnType<typeof renderMarkdown>>, TError = ErrorType<ErrorBody>>(
 markdownRequest: MarkdownRequest, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof renderMarkdown>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof renderMarkdown>>,
          TError,
          Awaited<ReturnType<typeof renderMarkdown>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any. This read-only operation remains available during maintenance.
 * @summary Preview sanitized Markdown
 */
export function useRenderMarkdown<TData = Awaited<ReturnType<typeof renderMarkdown>>, TError = ErrorType<ErrorBody>>(
 markdownRequest: MarkdownRequest, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof renderMarkdown>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof renderMarkdown>>,
          TError,
          Awaited<ReturnType<typeof renderMarkdown>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any. This read-only operation remains available during maintenance.
 * @summary Preview sanitized Markdown
 */
export function useRenderMarkdown<TData = Awaited<ReturnType<typeof renderMarkdown>>, TError = ErrorType<ErrorBody>>(
 markdownRequest: MarkdownRequest, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof renderMarkdown>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any. This read-only operation remains available during maintenance.
 * @summary Preview sanitized Markdown
 */
export function useRenderMarkdown<TData = Awaited<ReturnType<typeof renderMarkdown>>, TError = ErrorType<ErrorBody>>(
 markdownRequest: MarkdownRequest, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof renderMarkdown>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getRenderMarkdownQueryOptions(markdownRequest,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetMeUrl = () => {




  return `/api/me`
}

/**
 * Scope: any.
 * @summary The caller's identity
 */
export const getMe = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<MeView> => {

  return orvalFetch<MeView>(getGetMeUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetMeQueryKey = () => {
    return [
    `/api/me`
    ] as const;
    }


export const getGetMeQueryOptions = <TData = Awaited<ReturnType<typeof getMe>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getMe>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetMeQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getMe>>> = ({ signal }) => getMe({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getMe>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetMeQueryResult = NonNullable<Awaited<ReturnType<typeof getMe>>>
export type GetMeQueryError = ErrorType<ErrorBody>


/**
 * Scope: any.
 * @summary The caller's identity
 */
export function useGetMe<TData = Awaited<ReturnType<typeof getMe>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getMe>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getMe>>,
          TError,
          Awaited<ReturnType<typeof getMe>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary The caller's identity
 */
export function useGetMe<TData = Awaited<ReturnType<typeof getMe>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getMe>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getMe>>,
          TError,
          Awaited<ReturnType<typeof getMe>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary The caller's identity
 */
export function useGetMe<TData = Awaited<ReturnType<typeof getMe>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getMe>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary The caller's identity
 */
export function useGetMe<TData = Awaited<ReturnType<typeof getMe>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getMe>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetMeQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListOwnersUrl = () => {




  return `/api/owners`
}

/**
 * Scope: any.
 * @summary List owners with the repositories the caller can read
 */
export const listOwners = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<OwnerGroup[]> => {

  return orvalFetch<OwnerGroup[]>(getListOwnersUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListOwnersQueryKey = () => {
    return [
    `/api/owners`
    ] as const;
    }


export const getListOwnersQueryOptions = <TData = Awaited<ReturnType<typeof listOwners>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listOwners>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListOwnersQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listOwners>>> = ({ signal }) => listOwners({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listOwners>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListOwnersQueryResult = NonNullable<Awaited<ReturnType<typeof listOwners>>>
export type ListOwnersQueryError = ErrorType<ErrorBody>


/**
 * Scope: any.
 * @summary List owners with the repositories the caller can read
 */
export function useListOwners<TData = Awaited<ReturnType<typeof listOwners>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listOwners>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listOwners>>,
          TError,
          Awaited<ReturnType<typeof listOwners>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary List owners with the repositories the caller can read
 */
export function useListOwners<TData = Awaited<ReturnType<typeof listOwners>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listOwners>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listOwners>>,
          TError,
          Awaited<ReturnType<typeof listOwners>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary List owners with the repositories the caller can read
 */
export function useListOwners<TData = Awaited<ReturnType<typeof listOwners>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listOwners>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary List owners with the repositories the caller can read
 */
export function useListOwners<TData = Awaited<ReturnType<typeof listOwners>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listOwners>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListOwnersQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateOwnerUrl = () => {




  return `/api/owners`
}

/**
 * Scope: admin, unrestricted.
 * @summary Create an owner
 */
export const createOwner = async (createOwnerRequest: CreateOwnerRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<OwnerWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<OwnerWrite>(getCreateOwnerUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createOwnerRequest)
  }
);}





export const getCreateOwnerMutationKey = () => ['createOwner'] as const;

export const getCreateOwnerMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createOwner>>, TError,CreateOwnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createOwner>>, TError,CreateOwnerMutationVariables, TContext> => {

const mutationKey = getCreateOwnerMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createOwner>>, CreateOwnerMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  createOwner(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateOwnerMutationResult = NonNullable<Awaited<ReturnType<typeof createOwner>>>
    export type CreateOwnerMutationBody = CreateOwnerRequest
    export type CreateOwnerMutationError = ErrorType<ErrorBody>
    export type CreateOwnerMutationVariables = {data: CreateOwnerRequest}

    /**
 * Scope: admin, unrestricted.
 * @summary Create an owner
 */
export const useCreateOwner = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createOwner>>, TError,CreateOwnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createOwner>>,
        TError,
        CreateOwnerMutationVariables,
        TContext
      > => {
      return useMutation(getCreateOwnerMutationOptions(options), queryClient);
    }

export const getDeleteOwnerUrl = (owner: string,) => {




  return `/api/owners/${encodeURIComponent(String(owner))}`
}

/**
 * Scope: admin, unrestricted.
 * @summary Delete an owner that has no repositories
 */
export const deleteOwner = async (owner: string, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedWithEvent> => {

  return orvalFetch<DeletedWithEvent>(getDeleteOwnerUrl(owner),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteOwnerMutationKey = () => ['deleteOwner'] as const;

export const getDeleteOwnerMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteOwner>>, TError,DeleteOwnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteOwner>>, TError,DeleteOwnerMutationVariables, TContext> => {

const mutationKey = getDeleteOwnerMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteOwner>>, DeleteOwnerMutationVariables> = (props) => {
          const {owner} = props ?? {};

          return  deleteOwner(owner,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteOwnerMutationResult = NonNullable<Awaited<ReturnType<typeof deleteOwner>>>

    export type DeleteOwnerMutationError = ErrorType<ErrorBody>
    export type DeleteOwnerMutationVariables = {owner: string}

    /**
 * Scope: admin, unrestricted.
 * @summary Delete an owner that has no repositories
 */
export const useDeleteOwner = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteOwner>>, TError,DeleteOwnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteOwner>>,
        TError,
        DeleteOwnerMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteOwnerMutationOptions(options), queryClient);
    }

export const getGetOwnerUrl = (owner: string,) => {




  return `/api/owners/${encodeURIComponent(String(owner))}`
}

/**
 * Scope: any.
 * @summary Get an owner with its repositories
 */
export const getOwner = async (owner: string, options?: Parameters<typeof orvalFetch>[1]): Promise<OwnerGroup> => {

  return orvalFetch<OwnerGroup>(getGetOwnerUrl(owner),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetOwnerQueryKey = (owner: string,) => {
    return [
    `/api/owners/${owner}`
    ] as const;
    }


export const getGetOwnerQueryOptions = <TData = Awaited<ReturnType<typeof getOwner>>, TError = ErrorType<ErrorBody>>(owner: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getOwner>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetOwnerQueryKey(owner);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getOwner>>> = ({ signal }) => getOwner(owner, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getOwner>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetOwnerQueryResult = NonNullable<Awaited<ReturnType<typeof getOwner>>>
export type GetOwnerQueryError = ErrorType<ErrorBody>


/**
 * Scope: any.
 * @summary Get an owner with its repositories
 */
export function useGetOwner<TData = Awaited<ReturnType<typeof getOwner>>, TError = ErrorType<ErrorBody>>(
 owner: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getOwner>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getOwner>>,
          TError,
          Awaited<ReturnType<typeof getOwner>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary Get an owner with its repositories
 */
export function useGetOwner<TData = Awaited<ReturnType<typeof getOwner>>, TError = ErrorType<ErrorBody>>(
 owner: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getOwner>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getOwner>>,
          TError,
          Awaited<ReturnType<typeof getOwner>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary Get an owner with its repositories
 */
export function useGetOwner<TData = Awaited<ReturnType<typeof getOwner>>, TError = ErrorType<ErrorBody>>(
 owner: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getOwner>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any.
 * @summary Get an owner with its repositories
 */
export function useGetOwner<TData = Awaited<ReturnType<typeof getOwner>>, TError = ErrorType<ErrorBody>>(
 owner: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getOwner>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetOwnerQueryOptions(owner,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListReposUrl = () => {




  return `/api/repos`
}

/**
 * Scope: read.
 * @summary List the repositories the caller can read
 */
export const listRepos = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<RepoView[]> => {

  return orvalFetch<RepoView[]>(getListReposUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListReposQueryKey = () => {
    return [
    `/api/repos`
    ] as const;
    }


export const getListReposQueryOptions = <TData = Awaited<ReturnType<typeof listRepos>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepos>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListReposQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listRepos>>> = ({ signal }) => listRepos({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listRepos>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListReposQueryResult = NonNullable<Awaited<ReturnType<typeof listRepos>>>
export type ListReposQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List the repositories the caller can read
 */
export function useListRepos<TData = Awaited<ReturnType<typeof listRepos>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepos>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRepos>>,
          TError,
          Awaited<ReturnType<typeof listRepos>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List the repositories the caller can read
 */
export function useListRepos<TData = Awaited<ReturnType<typeof listRepos>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepos>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRepos>>,
          TError,
          Awaited<ReturnType<typeof listRepos>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List the repositories the caller can read
 */
export function useListRepos<TData = Awaited<ReturnType<typeof listRepos>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepos>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List the repositories the caller can read
 */
export function useListRepos<TData = Awaited<ReturnType<typeof listRepos>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepos>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListReposQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateRepoUrl = () => {




  return `/api/repos`
}

/**
 * Scope: admin. The owner must exist.
 * @summary Create a repository
 */
export const createRepo = async (createRepoRequest: CreateRepoRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<RepoView> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<RepoView>(getCreateRepoUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createRepoRequest)
  }
);}





export const getCreateRepoMutationKey = () => ['createRepo'] as const;

export const getCreateRepoMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRepo>>, TError,CreateRepoMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createRepo>>, TError,CreateRepoMutationVariables, TContext> => {

const mutationKey = getCreateRepoMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createRepo>>, CreateRepoMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  createRepo(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateRepoMutationResult = NonNullable<Awaited<ReturnType<typeof createRepo>>>
    export type CreateRepoMutationBody = CreateRepoRequest
    export type CreateRepoMutationError = ErrorType<ErrorBody>
    export type CreateRepoMutationVariables = {data: CreateRepoRequest}

    /**
 * Scope: admin. The owner must exist.
 * @summary Create a repository
 */
export const useCreateRepo = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRepo>>, TError,CreateRepoMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createRepo>>,
        TError,
        CreateRepoMutationVariables,
        TContext
      > => {
      return useMutation(getCreateRepoMutationOptions(options), queryClient);
    }

export const getDeleteRepoUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}`
}

/**
 * Scope: admin.
 * @summary Delete a repository with its releases and images
 */
export const deleteRepo = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedName> => {

  return orvalFetch<DeletedName>(getDeleteRepoUrl(owner,repo),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteRepoMutationKey = () => ['deleteRepo'] as const;

export const getDeleteRepoMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRepo>>, TError,DeleteRepoMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteRepo>>, TError,DeleteRepoMutationVariables, TContext> => {

const mutationKey = getDeleteRepoMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteRepo>>, DeleteRepoMutationVariables> = (props) => {
          const {owner,repo} = props ?? {};

          return  deleteRepo(owner,repo,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteRepoMutationResult = NonNullable<Awaited<ReturnType<typeof deleteRepo>>>

    export type DeleteRepoMutationError = ErrorType<ErrorBody>
    export type DeleteRepoMutationVariables = {owner: string;repo: string}

    /**
 * Scope: admin.
 * @summary Delete a repository with its releases and images
 */
export const useDeleteRepo = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRepo>>, TError,DeleteRepoMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteRepo>>,
        TError,
        DeleteRepoMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteRepoMutationOptions(options), queryClient);
    }

export const getGetRepoUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}`
}

/**
 * Scope: read.
 * @summary Get a repository
 */
export const getRepo = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<RepoView> => {

  return orvalFetch<RepoView>(getGetRepoUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetRepoQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}`
    ] as const;
    }


export const getGetRepoQueryOptions = <TData = Awaited<ReturnType<typeof getRepo>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepo>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetRepoQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getRepo>>> = ({ signal }) => getRepo(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getRepo>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetRepoQueryResult = NonNullable<Awaited<ReturnType<typeof getRepo>>>
export type GetRepoQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Get a repository
 */
export function useGetRepo<TData = Awaited<ReturnType<typeof getRepo>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepo>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepo>>,
          TError,
          Awaited<ReturnType<typeof getRepo>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a repository
 */
export function useGetRepo<TData = Awaited<ReturnType<typeof getRepo>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepo>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepo>>,
          TError,
          Awaited<ReturnType<typeof getRepo>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a repository
 */
export function useGetRepo<TData = Awaited<ReturnType<typeof getRepo>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepo>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a repository
 */
export function useGetRepo<TData = Awaited<ReturnType<typeof getRepo>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepo>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetRepoQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetRepositoryBlobUrl = (owner: string,
    repo: string,
    params?: GetRepositoryBlobParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/blob?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/blob`
}

/**
 * Scope: read.
 * @summary Read highlighted source or binary/large-file metadata
 */
export const getRepositoryBlob = async (owner: string,
    repo: string,
    params?: GetRepositoryBlobParams, options?: Parameters<typeof orvalFetch>[1]): Promise<BlobResponse> => {

  return orvalFetch<BlobResponse>(getGetRepositoryBlobUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetRepositoryBlobQueryKey = (owner: string,
    repo: string,
    params?: GetRepositoryBlobParams,) => {
    return [
    `/api/repos/${owner}/${repo}/blob`, ...(params ? [params] : [])
    ] as const;
    }


export const getGetRepositoryBlobQueryOptions = <TData = Awaited<ReturnType<typeof getRepositoryBlob>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: GetRepositoryBlobParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryBlob>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetRepositoryBlobQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getRepositoryBlob>>> = ({ signal }) => getRepositoryBlob(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getRepositoryBlob>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetRepositoryBlobQueryResult = NonNullable<Awaited<ReturnType<typeof getRepositoryBlob>>>
export type GetRepositoryBlobQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Read highlighted source or binary/large-file metadata
 */
export function useGetRepositoryBlob<TData = Awaited<ReturnType<typeof getRepositoryBlob>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  GetRepositoryBlobParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryBlob>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepositoryBlob>>,
          TError,
          Awaited<ReturnType<typeof getRepositoryBlob>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read highlighted source or binary/large-file metadata
 */
export function useGetRepositoryBlob<TData = Awaited<ReturnType<typeof getRepositoryBlob>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryBlobParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryBlob>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepositoryBlob>>,
          TError,
          Awaited<ReturnType<typeof getRepositoryBlob>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read highlighted source or binary/large-file metadata
 */
export function useGetRepositoryBlob<TData = Awaited<ReturnType<typeof getRepositoryBlob>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryBlobParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryBlob>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read highlighted source or binary/large-file metadata
 */
export function useGetRepositoryBlob<TData = Awaited<ReturnType<typeof getRepositoryBlob>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryBlobParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryBlob>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetRepositoryBlobQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListCommitsUrl = (owner: string,
    repo: string,
    params?: ListCommitsParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/commits?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/commits`
}

/**
 * Scope: read.
 * @summary List commits
 */
export const listCommits = async (owner: string,
    repo: string,
    params?: ListCommitsParams, options?: Parameters<typeof orvalFetch>[1]): Promise<Commit[]> => {

  return orvalFetch<Commit[]>(getListCommitsUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListCommitsQueryKey = (owner: string,
    repo: string,
    params?: ListCommitsParams,) => {
    return [
    `/api/repos/${owner}/${repo}/commits`, ...(params ? [params] : [])
    ] as const;
    }


export const getListCommitsQueryOptions = <TData = Awaited<ReturnType<typeof listCommits>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: ListCommitsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listCommits>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListCommitsQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listCommits>>> = ({ signal }) => listCommits(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listCommits>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListCommitsQueryResult = NonNullable<Awaited<ReturnType<typeof listCommits>>>
export type ListCommitsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List commits
 */
export function useListCommits<TData = Awaited<ReturnType<typeof listCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  ListCommitsParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listCommits>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listCommits>>,
          TError,
          Awaited<ReturnType<typeof listCommits>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List commits
 */
export function useListCommits<TData = Awaited<ReturnType<typeof listCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListCommitsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listCommits>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listCommits>>,
          TError,
          Awaited<ReturnType<typeof listCommits>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List commits
 */
export function useListCommits<TData = Awaited<ReturnType<typeof listCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListCommitsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listCommits>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List commits
 */
export function useListCommits<TData = Awaited<ReturnType<typeof listCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListCommitsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listCommits>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListCommitsQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetCommitUrl = (owner: string,
    repo: string,
    rev: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/commits/${encodeURIComponent(String(rev))}`
}

/**
 * Scope: read.
 * @summary Get a commit
 */
export const getCommit = async (owner: string,
    repo: string,
    rev: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Commit> => {

  return orvalFetch<Commit>(getGetCommitUrl(owner,repo,rev),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetCommitQueryKey = (owner: string,
    repo: string,
    rev: string,) => {
    return [
    `/api/repos/${owner}/${repo}/commits/${rev}`
    ] as const;
    }


export const getGetCommitQueryOptions = <TData = Awaited<ReturnType<typeof getCommit>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommit>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetCommitQueryKey(owner,repo,rev);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getCommit>>> = ({ signal }) => getCommit(owner,repo,rev, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && rev !== null && rev !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getCommit>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetCommitQueryResult = NonNullable<Awaited<ReturnType<typeof getCommit>>>
export type GetCommitQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Get a commit
 */
export function useGetCommit<TData = Awaited<ReturnType<typeof getCommit>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommit>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getCommit>>,
          TError,
          Awaited<ReturnType<typeof getCommit>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a commit
 */
export function useGetCommit<TData = Awaited<ReturnType<typeof getCommit>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommit>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getCommit>>,
          TError,
          Awaited<ReturnType<typeof getCommit>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a commit
 */
export function useGetCommit<TData = Awaited<ReturnType<typeof getCommit>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommit>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a commit
 */
export function useGetCommit<TData = Awaited<ReturnType<typeof getCommit>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommit>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetCommitQueryOptions(owner,repo,rev,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetCommitWorkspaceUrl = (owner: string,
    repo: string,
    rev: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/commits/${encodeURIComponent(String(rev))}/view`
}

/**
 * Scope: read.
 * @summary Read a commit with checks and structured diffs
 */
export const getCommitWorkspace = async (owner: string,
    repo: string,
    rev: string, options?: Parameters<typeof orvalFetch>[1]): Promise<CommitWorkspace> => {

  return orvalFetch<CommitWorkspace>(getGetCommitWorkspaceUrl(owner,repo,rev),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetCommitWorkspaceQueryKey = (owner: string,
    repo: string,
    rev: string,) => {
    return [
    `/api/repos/${owner}/${repo}/commits/${rev}/view`
    ] as const;
    }


export const getGetCommitWorkspaceQueryOptions = <TData = Awaited<ReturnType<typeof getCommitWorkspace>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommitWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetCommitWorkspaceQueryKey(owner,repo,rev);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getCommitWorkspace>>> = ({ signal }) => getCommitWorkspace(owner,repo,rev, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && rev !== null && rev !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getCommitWorkspace>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetCommitWorkspaceQueryResult = NonNullable<Awaited<ReturnType<typeof getCommitWorkspace>>>
export type GetCommitWorkspaceQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Read a commit with checks and structured diffs
 */
export function useGetCommitWorkspace<TData = Awaited<ReturnType<typeof getCommitWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommitWorkspace>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getCommitWorkspace>>,
          TError,
          Awaited<ReturnType<typeof getCommitWorkspace>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read a commit with checks and structured diffs
 */
export function useGetCommitWorkspace<TData = Awaited<ReturnType<typeof getCommitWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommitWorkspace>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getCommitWorkspace>>,
          TError,
          Awaited<ReturnType<typeof getCommitWorkspace>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read a commit with checks and structured diffs
 */
export function useGetCommitWorkspace<TData = Awaited<ReturnType<typeof getCommitWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommitWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read a commit with checks and structured diffs
 */
export function useGetCommitWorkspace<TData = Awaited<ReturnType<typeof getCommitWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    rev: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getCommitWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetCommitWorkspaceQueryOptions(owner,repo,rev,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListRepositoryFilesUrl = (owner: string,
    repo: string,
    params?: ListRepositoryFilesParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/files?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/files`
}

/**
 * Scope: read.
 * @summary List recursive file paths for the file finder
 */
export const listRepositoryFiles = async (owner: string,
    repo: string,
    params?: ListRepositoryFilesParams, options?: Parameters<typeof orvalFetch>[1]): Promise<string[]> => {

  return orvalFetch<string[]>(getListRepositoryFilesUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListRepositoryFilesQueryKey = (owner: string,
    repo: string,
    params?: ListRepositoryFilesParams,) => {
    return [
    `/api/repos/${owner}/${repo}/files`, ...(params ? [params] : [])
    ] as const;
    }


export const getListRepositoryFilesQueryOptions = <TData = Awaited<ReturnType<typeof listRepositoryFiles>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: ListRepositoryFilesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepositoryFiles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListRepositoryFilesQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listRepositoryFiles>>> = ({ signal }) => listRepositoryFiles(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listRepositoryFiles>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListRepositoryFilesQueryResult = NonNullable<Awaited<ReturnType<typeof listRepositoryFiles>>>
export type ListRepositoryFilesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List recursive file paths for the file finder
 */
export function useListRepositoryFiles<TData = Awaited<ReturnType<typeof listRepositoryFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  ListRepositoryFilesParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepositoryFiles>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRepositoryFiles>>,
          TError,
          Awaited<ReturnType<typeof listRepositoryFiles>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List recursive file paths for the file finder
 */
export function useListRepositoryFiles<TData = Awaited<ReturnType<typeof listRepositoryFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListRepositoryFilesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepositoryFiles>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRepositoryFiles>>,
          TError,
          Awaited<ReturnType<typeof listRepositoryFiles>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List recursive file paths for the file finder
 */
export function useListRepositoryFiles<TData = Awaited<ReturnType<typeof listRepositoryFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListRepositoryFilesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepositoryFiles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List recursive file paths for the file finder
 */
export function useListRepositoryFiles<TData = Awaited<ReturnType<typeof listRepositoryFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListRepositoryFilesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRepositoryFiles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListRepositoryFilesQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListImagesUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/images`
}

/**
 * Scope: read. Images are pushed and pulled through the OCI distribution API at /v2/.
 * @summary List container image tags
 */
export const listImages = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Image[]> => {

  return orvalFetch<Image[]>(getListImagesUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListImagesQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/images`
    ] as const;
    }


export const getListImagesQueryOptions = <TData = Awaited<ReturnType<typeof listImages>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listImages>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListImagesQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listImages>>> = ({ signal }) => listImages(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listImages>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListImagesQueryResult = NonNullable<Awaited<ReturnType<typeof listImages>>>
export type ListImagesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read. Images are pushed and pulled through the OCI distribution API at /v2/.
 * @summary List container image tags
 */
export function useListImages<TData = Awaited<ReturnType<typeof listImages>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listImages>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listImages>>,
          TError,
          Awaited<ReturnType<typeof listImages>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Images are pushed and pulled through the OCI distribution API at /v2/.
 * @summary List container image tags
 */
export function useListImages<TData = Awaited<ReturnType<typeof listImages>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listImages>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listImages>>,
          TError,
          Awaited<ReturnType<typeof listImages>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Images are pushed and pulled through the OCI distribution API at /v2/.
 * @summary List container image tags
 */
export function useListImages<TData = Awaited<ReturnType<typeof listImages>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listImages>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Images are pushed and pulled through the OCI distribution API at /v2/.
 * @summary List container image tags
 */
export function useListImages<TData = Awaited<ReturnType<typeof listImages>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listImages>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListImagesQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListIssuesUrl = (owner: string,
    repo: string,
    params?: ListIssuesParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    const explodeParameters = ["label","assignee"];

    if (Array.isArray(value) && explodeParameters.includes(key)) {
      value.forEach((v) => {
        normalizedParams.append(key, v === null ? 'null' : String(v));
      });
      return;
    }

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues`
}

/**
 * Scope: read.
 * @summary List issues
 */
export const listIssues = async (owner: string,
    repo: string,
    params?: ListIssuesParams, options?: Parameters<typeof orvalFetch>[1]): Promise<Item[]> => {

  return orvalFetch<Item[]>(getListIssuesUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListIssuesQueryKey = (owner: string,
    repo: string,
    params?: ListIssuesParams,) => {
    return [
    `/api/repos/${owner}/${repo}/issues`, ...(params ? [params] : [])
    ] as const;
    }


export const getListIssuesQueryOptions = <TData = Awaited<ReturnType<typeof listIssues>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: ListIssuesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listIssues>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListIssuesQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listIssues>>> = ({ signal }) => listIssues(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listIssues>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListIssuesQueryResult = NonNullable<Awaited<ReturnType<typeof listIssues>>>
export type ListIssuesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List issues
 */
export function useListIssues<TData = Awaited<ReturnType<typeof listIssues>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  ListIssuesParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listIssues>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listIssues>>,
          TError,
          Awaited<ReturnType<typeof listIssues>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List issues
 */
export function useListIssues<TData = Awaited<ReturnType<typeof listIssues>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListIssuesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listIssues>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listIssues>>,
          TError,
          Awaited<ReturnType<typeof listIssues>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List issues
 */
export function useListIssues<TData = Awaited<ReturnType<typeof listIssues>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListIssuesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listIssues>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List issues
 */
export function useListIssues<TData = Awaited<ReturnType<typeof listIssues>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListIssuesParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listIssues>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListIssuesQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateIssueUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues`
}

/**
 * Scope: write.
 * @summary Open an issue
 */
export const createIssue = async (owner: string,
    repo: string,
    createIssueRequest: CreateIssueRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<ItemWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<ItemWrite>(getCreateIssueUrl(owner,repo),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createIssueRequest)
  }
);}





export const getCreateIssueMutationKey = () => ['createIssue'] as const;

export const getCreateIssueMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createIssue>>, TError,CreateIssueMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createIssue>>, TError,CreateIssueMutationVariables, TContext> => {

const mutationKey = getCreateIssueMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createIssue>>, CreateIssueMutationVariables> = (props) => {
          const {owner,repo,data} = props ?? {};

          return  createIssue(owner,repo,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateIssueMutationResult = NonNullable<Awaited<ReturnType<typeof createIssue>>>
    export type CreateIssueMutationBody = CreateIssueRequest
    export type CreateIssueMutationError = ErrorType<ErrorBody>
    export type CreateIssueMutationVariables = {owner: string;repo: string;data: CreateIssueRequest}

    /**
 * Scope: write.
 * @summary Open an issue
 */
export const useCreateIssue = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createIssue>>, TError,CreateIssueMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createIssue>>,
        TError,
        CreateIssueMutationVariables,
        TContext
      > => {
      return useMutation(getCreateIssueMutationOptions(options), queryClient);
    }

export const getGetIssueUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues/${encodeURIComponent(String(number))}`
}

/**
 * Scope: read.
 * @summary Get an issue with its comments
 */
export const getIssue = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<ItemView> => {

  return orvalFetch<ItemView>(getGetIssueUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetIssueQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/issues/${number}`
    ] as const;
    }


export const getGetIssueQueryOptions = <TData = Awaited<ReturnType<typeof getIssue>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssue>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetIssueQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getIssue>>> = ({ signal }) => getIssue(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getIssue>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetIssueQueryResult = NonNullable<Awaited<ReturnType<typeof getIssue>>>
export type GetIssueQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Get an issue with its comments
 */
export function useGetIssue<TData = Awaited<ReturnType<typeof getIssue>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssue>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getIssue>>,
          TError,
          Awaited<ReturnType<typeof getIssue>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get an issue with its comments
 */
export function useGetIssue<TData = Awaited<ReturnType<typeof getIssue>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssue>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getIssue>>,
          TError,
          Awaited<ReturnType<typeof getIssue>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get an issue with its comments
 */
export function useGetIssue<TData = Awaited<ReturnType<typeof getIssue>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssue>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get an issue with its comments
 */
export function useGetIssue<TData = Awaited<ReturnType<typeof getIssue>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssue>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetIssueQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getEditIssueUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues/${encodeURIComponent(String(number))}`
}

/**
 * Scope: write. Each change is recorded as its own event; event_id is the last one.
 * @summary Edit an issue
 */
export const editIssue = async (owner: string,
    repo: string,
    number: number,
    editItemRequest: EditItemRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<ItemWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<ItemWrite>(getEditIssueUrl(owner,repo,number),
  {
    ...options,
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(editItemRequest)
  }
);}





export const getEditIssueMutationKey = () => ['editIssue'] as const;

export const getEditIssueMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof editIssue>>, TError,EditIssueMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof editIssue>>, TError,EditIssueMutationVariables, TContext> => {

const mutationKey = getEditIssueMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof editIssue>>, EditIssueMutationVariables> = (props) => {
          const {owner,repo,number,data} = props ?? {};

          return  editIssue(owner,repo,number,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type EditIssueMutationResult = NonNullable<Awaited<ReturnType<typeof editIssue>>>
    export type EditIssueMutationBody = EditItemRequest
    export type EditIssueMutationError = ErrorType<ErrorBody>
    export type EditIssueMutationVariables = {owner: string;repo: string;number: number;data: EditItemRequest}

    /**
 * Scope: write. Each change is recorded as its own event; event_id is the last one.
 * @summary Edit an issue
 */
export const useEditIssue = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof editIssue>>, TError,EditIssueMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof editIssue>>,
        TError,
        EditIssueMutationVariables,
        TContext
      > => {
      return useMutation(getEditIssueMutationOptions(options), queryClient);
    }

export const getCommentIssueUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues/${encodeURIComponent(String(number))}/comments`
}

/**
 * Scope: write.
 * @summary Comment on an issue
 */
export const commentIssue = async (owner: string,
    repo: string,
    number: number,
    commentRequest: CommentRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<CommentWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<CommentWrite>(getCommentIssueUrl(owner,repo,number),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(commentRequest)
  }
);}





export const getCommentIssueMutationKey = () => ['commentIssue'] as const;

export const getCommentIssueMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof commentIssue>>, TError,CommentIssueMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof commentIssue>>, TError,CommentIssueMutationVariables, TContext> => {

const mutationKey = getCommentIssueMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof commentIssue>>, CommentIssueMutationVariables> = (props) => {
          const {owner,repo,number,data} = props ?? {};

          return  commentIssue(owner,repo,number,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CommentIssueMutationResult = NonNullable<Awaited<ReturnType<typeof commentIssue>>>
    export type CommentIssueMutationBody = CommentRequest
    export type CommentIssueMutationError = ErrorType<ErrorBody>
    export type CommentIssueMutationVariables = {owner: string;repo: string;number: number;data: CommentRequest}

    /**
 * Scope: write.
 * @summary Comment on an issue
 */
export const useCommentIssue = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof commentIssue>>, TError,CommentIssueMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof commentIssue>>,
        TError,
        CommentIssueMutationVariables,
        TContext
      > => {
      return useMutation(getCommentIssueMutationOptions(options), queryClient);
    }

export const getGetIssueWorkspaceUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/issues/${encodeURIComponent(String(number))}/view`
}

/**
 * Scope: read.
 * @summary Read an issue with rendered discussion and permissions
 */
export const getIssueWorkspace = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<ItemWorkspace> => {

  return orvalFetch<ItemWorkspace>(getGetIssueWorkspaceUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetIssueWorkspaceQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/issues/${number}/view`
    ] as const;
    }


export const getGetIssueWorkspaceQueryOptions = <TData = Awaited<ReturnType<typeof getIssueWorkspace>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssueWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetIssueWorkspaceQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getIssueWorkspace>>> = ({ signal }) => getIssueWorkspace(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getIssueWorkspace>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetIssueWorkspaceQueryResult = NonNullable<Awaited<ReturnType<typeof getIssueWorkspace>>>
export type GetIssueWorkspaceQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Read an issue with rendered discussion and permissions
 */
export function useGetIssueWorkspace<TData = Awaited<ReturnType<typeof getIssueWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssueWorkspace>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getIssueWorkspace>>,
          TError,
          Awaited<ReturnType<typeof getIssueWorkspace>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read an issue with rendered discussion and permissions
 */
export function useGetIssueWorkspace<TData = Awaited<ReturnType<typeof getIssueWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssueWorkspace>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getIssueWorkspace>>,
          TError,
          Awaited<ReturnType<typeof getIssueWorkspace>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read an issue with rendered discussion and permissions
 */
export function useGetIssueWorkspace<TData = Awaited<ReturnType<typeof getIssueWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssueWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read an issue with rendered discussion and permissions
 */
export function useGetIssueWorkspace<TData = Awaited<ReturnType<typeof getIssueWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getIssueWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetIssueWorkspaceQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListLabelsUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/labels`
}

/**
 * Scope: read.
 * @summary List labels
 */
export const listLabels = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Label[]> => {

  return orvalFetch<Label[]>(getListLabelsUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListLabelsQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/labels`
    ] as const;
    }


export const getListLabelsQueryOptions = <TData = Awaited<ReturnType<typeof listLabels>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listLabels>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListLabelsQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listLabels>>> = ({ signal }) => listLabels(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listLabels>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListLabelsQueryResult = NonNullable<Awaited<ReturnType<typeof listLabels>>>
export type ListLabelsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List labels
 */
export function useListLabels<TData = Awaited<ReturnType<typeof listLabels>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listLabels>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listLabels>>,
          TError,
          Awaited<ReturnType<typeof listLabels>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List labels
 */
export function useListLabels<TData = Awaited<ReturnType<typeof listLabels>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listLabels>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listLabels>>,
          TError,
          Awaited<ReturnType<typeof listLabels>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List labels
 */
export function useListLabels<TData = Awaited<ReturnType<typeof listLabels>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listLabels>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List labels
 */
export function useListLabels<TData = Awaited<ReturnType<typeof listLabels>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listLabels>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListLabelsQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateLabelUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/labels`
}

/**
 * Scope: write.
 * @summary Create a label
 */
export const createLabel = async (owner: string,
    repo: string,
    createLabelRequest: CreateLabelRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<LabelWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<LabelWrite>(getCreateLabelUrl(owner,repo),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createLabelRequest)
  }
);}





export const getCreateLabelMutationKey = () => ['createLabel'] as const;

export const getCreateLabelMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createLabel>>, TError,CreateLabelMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createLabel>>, TError,CreateLabelMutationVariables, TContext> => {

const mutationKey = getCreateLabelMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createLabel>>, CreateLabelMutationVariables> = (props) => {
          const {owner,repo,data} = props ?? {};

          return  createLabel(owner,repo,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateLabelMutationResult = NonNullable<Awaited<ReturnType<typeof createLabel>>>
    export type CreateLabelMutationBody = CreateLabelRequest
    export type CreateLabelMutationError = ErrorType<ErrorBody>
    export type CreateLabelMutationVariables = {owner: string;repo: string;data: CreateLabelRequest}

    /**
 * Scope: write.
 * @summary Create a label
 */
export const useCreateLabel = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createLabel>>, TError,CreateLabelMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createLabel>>,
        TError,
        CreateLabelMutationVariables,
        TContext
      > => {
      return useMutation(getCreateLabelMutationOptions(options), queryClient);
    }

export const getDeleteLabelUrl = (owner: string,
    repo: string,
    label: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/labels/${encodeURIComponent(String(label))}`
}

/**
 * Scope: write.
 * @summary Delete a label
 */
export const deleteLabel = async (owner: string,
    repo: string,
    label: string, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedWithEvent> => {

  return orvalFetch<DeletedWithEvent>(getDeleteLabelUrl(owner,repo,label),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteLabelMutationKey = () => ['deleteLabel'] as const;

export const getDeleteLabelMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteLabel>>, TError,DeleteLabelMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteLabel>>, TError,DeleteLabelMutationVariables, TContext> => {

const mutationKey = getDeleteLabelMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteLabel>>, DeleteLabelMutationVariables> = (props) => {
          const {owner,repo,label} = props ?? {};

          return  deleteLabel(owner,repo,label,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteLabelMutationResult = NonNullable<Awaited<ReturnType<typeof deleteLabel>>>

    export type DeleteLabelMutationError = ErrorType<ErrorBody>
    export type DeleteLabelMutationVariables = {owner: string;repo: string;label: string}

    /**
 * Scope: write.
 * @summary Delete a label
 */
export const useDeleteLabel = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteLabel>>, TError,DeleteLabelMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteLabel>>,
        TError,
        DeleteLabelMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteLabelMutationOptions(options), queryClient);
    }

export const getListPullsUrl = (owner: string,
    repo: string,
    params?: ListPullsParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {
    const explodeParameters = ["label","assignee"];

    if (Array.isArray(value) && explodeParameters.includes(key)) {
      value.forEach((v) => {
        normalizedParams.append(key, v === null ? 'null' : String(v));
      });
      return;
    }

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls`
}

/**
 * Scope: read.
 * @summary List pull requests
 */
export const listPulls = async (owner: string,
    repo: string,
    params?: ListPullsParams, options?: Parameters<typeof orvalFetch>[1]): Promise<Pull[]> => {

  return orvalFetch<Pull[]>(getListPullsUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListPullsQueryKey = (owner: string,
    repo: string,
    params?: ListPullsParams,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls`, ...(params ? [params] : [])
    ] as const;
    }


export const getListPullsQueryOptions = <TData = Awaited<ReturnType<typeof listPulls>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: ListPullsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPulls>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListPullsQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listPulls>>> = ({ signal }) => listPulls(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listPulls>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListPullsQueryResult = NonNullable<Awaited<ReturnType<typeof listPulls>>>
export type ListPullsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List pull requests
 */
export function useListPulls<TData = Awaited<ReturnType<typeof listPulls>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  ListPullsParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPulls>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listPulls>>,
          TError,
          Awaited<ReturnType<typeof listPulls>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List pull requests
 */
export function useListPulls<TData = Awaited<ReturnType<typeof listPulls>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListPullsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPulls>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listPulls>>,
          TError,
          Awaited<ReturnType<typeof listPulls>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List pull requests
 */
export function useListPulls<TData = Awaited<ReturnType<typeof listPulls>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListPullsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPulls>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List pull requests
 */
export function useListPulls<TData = Awaited<ReturnType<typeof listPulls>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListPullsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPulls>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListPullsQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreatePullUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls`
}

/**
 * Scope: write. actions lists the Actions runs it queued.
 * @summary Open a pull request
 */
export const createPull = async (owner: string,
    repo: string,
    createPullRequest: CreatePullRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<PullWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<PullWrite>(getCreatePullUrl(owner,repo),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createPullRequest)
  }
);}





export const getCreatePullMutationKey = () => ['createPull'] as const;

export const getCreatePullMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createPull>>, TError,CreatePullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createPull>>, TError,CreatePullMutationVariables, TContext> => {

const mutationKey = getCreatePullMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createPull>>, CreatePullMutationVariables> = (props) => {
          const {owner,repo,data} = props ?? {};

          return  createPull(owner,repo,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreatePullMutationResult = NonNullable<Awaited<ReturnType<typeof createPull>>>
    export type CreatePullMutationBody = CreatePullRequest
    export type CreatePullMutationError = ErrorType<ErrorBody>
    export type CreatePullMutationVariables = {owner: string;repo: string;data: CreatePullRequest}

    /**
 * Scope: write. actions lists the Actions runs it queued.
 * @summary Open a pull request
 */
export const useCreatePull = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createPull>>, TError,CreatePullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createPull>>,
        TError,
        CreatePullMutationVariables,
        TContext
      > => {
      return useMutation(getCreatePullMutationOptions(options), queryClient);
    }

export const getGetPullUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}`
}

/**
 * Scope: read.
 * @summary Get a pull request with its merge status, reviews, comments and checks
 */
export const getPull = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<PullView> => {

  return orvalFetch<PullView>(getGetPullUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetPullQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls/${number}`
    ] as const;
    }


export const getGetPullQueryOptions = <TData = Awaited<ReturnType<typeof getPull>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPull>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetPullQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getPull>>> = ({ signal }) => getPull(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getPull>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetPullQueryResult = NonNullable<Awaited<ReturnType<typeof getPull>>>
export type GetPullQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Get a pull request with its merge status, reviews, comments and checks
 */
export function useGetPull<TData = Awaited<ReturnType<typeof getPull>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPull>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPull>>,
          TError,
          Awaited<ReturnType<typeof getPull>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a pull request with its merge status, reviews, comments and checks
 */
export function useGetPull<TData = Awaited<ReturnType<typeof getPull>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPull>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPull>>,
          TError,
          Awaited<ReturnType<typeof getPull>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a pull request with its merge status, reviews, comments and checks
 */
export function useGetPull<TData = Awaited<ReturnType<typeof getPull>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPull>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a pull request with its merge status, reviews, comments and checks
 */
export function useGetPull<TData = Awaited<ReturnType<typeof getPull>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPull>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetPullQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getEditPullUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}`
}

/**
 * Scope: write. Each change is recorded as its own event; event_id is the last one. Reopening syncs the pull request with its head branch, and is refused if that branch was deleted.
 * @summary Edit a pull request
 */
export const editPull = async (owner: string,
    repo: string,
    number: number,
    editItemRequest: EditItemRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<PullWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<PullWrite>(getEditPullUrl(owner,repo,number),
  {
    ...options,
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(editItemRequest)
  }
);}





export const getEditPullMutationKey = () => ['editPull'] as const;

export const getEditPullMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof editPull>>, TError,EditPullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof editPull>>, TError,EditPullMutationVariables, TContext> => {

const mutationKey = getEditPullMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof editPull>>, EditPullMutationVariables> = (props) => {
          const {owner,repo,number,data} = props ?? {};

          return  editPull(owner,repo,number,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type EditPullMutationResult = NonNullable<Awaited<ReturnType<typeof editPull>>>
    export type EditPullMutationBody = EditItemRequest
    export type EditPullMutationError = ErrorType<ErrorBody>
    export type EditPullMutationVariables = {owner: string;repo: string;number: number;data: EditItemRequest}

    /**
 * Scope: write. Each change is recorded as its own event; event_id is the last one. Reopening syncs the pull request with its head branch, and is refused if that branch was deleted.
 * @summary Edit a pull request
 */
export const useEditPull = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof editPull>>, TError,EditPullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof editPull>>,
        TError,
        EditPullMutationVariables,
        TContext
      > => {
      return useMutation(getEditPullMutationOptions(options), queryClient);
    }

export const getCommentPullUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/comments`
}

/**
 * Scope: write.
 * @summary Comment on a pull request, optionally on a line of its diff
 */
export const commentPull = async (owner: string,
    repo: string,
    number: number,
    commentRequest: CommentRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<CommentWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<CommentWrite>(getCommentPullUrl(owner,repo,number),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(commentRequest)
  }
);}





export const getCommentPullMutationKey = () => ['commentPull'] as const;

export const getCommentPullMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof commentPull>>, TError,CommentPullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof commentPull>>, TError,CommentPullMutationVariables, TContext> => {

const mutationKey = getCommentPullMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof commentPull>>, CommentPullMutationVariables> = (props) => {
          const {owner,repo,number,data} = props ?? {};

          return  commentPull(owner,repo,number,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CommentPullMutationResult = NonNullable<Awaited<ReturnType<typeof commentPull>>>
    export type CommentPullMutationBody = CommentRequest
    export type CommentPullMutationError = ErrorType<ErrorBody>
    export type CommentPullMutationVariables = {owner: string;repo: string;number: number;data: CommentRequest}

    /**
 * Scope: write.
 * @summary Comment on a pull request, optionally on a line of its diff
 */
export const useCommentPull = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof commentPull>>, TError,CommentPullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof commentPull>>,
        TError,
        CommentPullMutationVariables,
        TContext
      > => {
      return useMutation(getCommentPullMutationOptions(options), queryClient);
    }

export const getListPullCommitsUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/commits`
}

/**
 * Scope: read.
 * @summary List a pull request's commits
 */
export const listPullCommits = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<Commit[]> => {

  return orvalFetch<Commit[]>(getListPullCommitsUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListPullCommitsQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls/${number}/commits`
    ] as const;
    }


export const getListPullCommitsQueryOptions = <TData = Awaited<ReturnType<typeof listPullCommits>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullCommits>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListPullCommitsQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listPullCommits>>> = ({ signal }) => listPullCommits(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listPullCommits>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListPullCommitsQueryResult = NonNullable<Awaited<ReturnType<typeof listPullCommits>>>
export type ListPullCommitsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List a pull request's commits
 */
export function useListPullCommits<TData = Awaited<ReturnType<typeof listPullCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullCommits>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listPullCommits>>,
          TError,
          Awaited<ReturnType<typeof listPullCommits>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List a pull request's commits
 */
export function useListPullCommits<TData = Awaited<ReturnType<typeof listPullCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullCommits>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listPullCommits>>,
          TError,
          Awaited<ReturnType<typeof listPullCommits>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List a pull request's commits
 */
export function useListPullCommits<TData = Awaited<ReturnType<typeof listPullCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullCommits>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List a pull request's commits
 */
export function useListPullCommits<TData = Awaited<ReturnType<typeof listPullCommits>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullCommits>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListPullCommitsQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetPullDiffUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/diff`
}

/**
 * Scope: read.
 * @summary The unified diff against the merge base
 */
export const getPullDiff = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<string> => {

  return orvalFetch<string>(getGetPullDiffUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetPullDiffQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls/${number}/diff`
    ] as const;
    }


export const getGetPullDiffQueryOptions = <TData = Awaited<ReturnType<typeof getPullDiff>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullDiff>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetPullDiffQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getPullDiff>>> = ({ signal }) => getPullDiff(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getPullDiff>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetPullDiffQueryResult = NonNullable<Awaited<ReturnType<typeof getPullDiff>>>
export type GetPullDiffQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary The unified diff against the merge base
 */
export function useGetPullDiff<TData = Awaited<ReturnType<typeof getPullDiff>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullDiff>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPullDiff>>,
          TError,
          Awaited<ReturnType<typeof getPullDiff>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary The unified diff against the merge base
 */
export function useGetPullDiff<TData = Awaited<ReturnType<typeof getPullDiff>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullDiff>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPullDiff>>,
          TError,
          Awaited<ReturnType<typeof getPullDiff>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary The unified diff against the merge base
 */
export function useGetPullDiff<TData = Awaited<ReturnType<typeof getPullDiff>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullDiff>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary The unified diff against the merge base
 */
export function useGetPullDiff<TData = Awaited<ReturnType<typeof getPullDiff>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullDiff>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetPullDiffQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListPullFilesUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/files`
}

/**
 * Scope: read.
 * @summary List changed files
 */
export const listPullFiles = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<FileChange[]> => {

  return orvalFetch<FileChange[]>(getListPullFilesUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListPullFilesQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls/${number}/files`
    ] as const;
    }


export const getListPullFilesQueryOptions = <TData = Awaited<ReturnType<typeof listPullFiles>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullFiles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListPullFilesQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listPullFiles>>> = ({ signal }) => listPullFiles(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listPullFiles>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListPullFilesQueryResult = NonNullable<Awaited<ReturnType<typeof listPullFiles>>>
export type ListPullFilesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List changed files
 */
export function useListPullFiles<TData = Awaited<ReturnType<typeof listPullFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullFiles>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listPullFiles>>,
          TError,
          Awaited<ReturnType<typeof listPullFiles>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List changed files
 */
export function useListPullFiles<TData = Awaited<ReturnType<typeof listPullFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullFiles>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listPullFiles>>,
          TError,
          Awaited<ReturnType<typeof listPullFiles>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List changed files
 */
export function useListPullFiles<TData = Awaited<ReturnType<typeof listPullFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullFiles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List changed files
 */
export function useListPullFiles<TData = Awaited<ReturnType<typeof listPullFiles>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listPullFiles>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListPullFilesQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getMergePullUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/merge`
}

/**
 * Scope: write. Returns 409 on conflicts, when a branch moved, or when head_sha doesn't match the head branch.
 * @summary Merge a pull request
 */
export const mergePull = async (owner: string,
    repo: string,
    number: number,
    mergeRequest: MergeRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<PullWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<PullWrite>(getMergePullUrl(owner,repo,number),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(mergeRequest)
  }
);}





export const getMergePullMutationKey = () => ['mergePull'] as const;

export const getMergePullMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof mergePull>>, TError,MergePullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof mergePull>>, TError,MergePullMutationVariables, TContext> => {

const mutationKey = getMergePullMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof mergePull>>, MergePullMutationVariables> = (props) => {
          const {owner,repo,number,data} = props ?? {};

          return  mergePull(owner,repo,number,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type MergePullMutationResult = NonNullable<Awaited<ReturnType<typeof mergePull>>>
    export type MergePullMutationBody = MergeRequest
    export type MergePullMutationError = ErrorType<ErrorBody>
    export type MergePullMutationVariables = {owner: string;repo: string;number: number;data: MergeRequest}

    /**
 * Scope: write. Returns 409 on conflicts, when a branch moved, or when head_sha doesn't match the head branch.
 * @summary Merge a pull request
 */
export const useMergePull = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof mergePull>>, TError,MergePullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof mergePull>>,
        TError,
        MergePullMutationVariables,
        TContext
      > => {
      return useMutation(getMergePullMutationOptions(options), queryClient);
    }

export const getGetPullPatchUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/patch`
}

/**
 * Scope: read.
 * @summary Read structured, highlighted diffs and anchored comments
 */
export const getPullPatch = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<DiffFile[]> => {

  return orvalFetch<DiffFile[]>(getGetPullPatchUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetPullPatchQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls/${number}/patch`
    ] as const;
    }


export const getGetPullPatchQueryOptions = <TData = Awaited<ReturnType<typeof getPullPatch>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullPatch>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetPullPatchQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getPullPatch>>> = ({ signal }) => getPullPatch(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getPullPatch>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetPullPatchQueryResult = NonNullable<Awaited<ReturnType<typeof getPullPatch>>>
export type GetPullPatchQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Read structured, highlighted diffs and anchored comments
 */
export function useGetPullPatch<TData = Awaited<ReturnType<typeof getPullPatch>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullPatch>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPullPatch>>,
          TError,
          Awaited<ReturnType<typeof getPullPatch>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read structured, highlighted diffs and anchored comments
 */
export function useGetPullPatch<TData = Awaited<ReturnType<typeof getPullPatch>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullPatch>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPullPatch>>,
          TError,
          Awaited<ReturnType<typeof getPullPatch>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read structured, highlighted diffs and anchored comments
 */
export function useGetPullPatch<TData = Awaited<ReturnType<typeof getPullPatch>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullPatch>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read structured, highlighted diffs and anchored comments
 */
export function useGetPullPatch<TData = Awaited<ReturnType<typeof getPullPatch>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullPatch>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetPullPatchQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getReviewPullUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/reviews`
}

/**
 * Scope: write. Authors can't approve or request changes on their own pull requests.
 * @summary Review a pull request's current head commit
 */
export const reviewPull = async (owner: string,
    repo: string,
    number: number,
    reviewRequest: ReviewRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<ReviewWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<ReviewWrite>(getReviewPullUrl(owner,repo,number),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(reviewRequest)
  }
);}





export const getReviewPullMutationKey = () => ['reviewPull'] as const;

export const getReviewPullMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof reviewPull>>, TError,ReviewPullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof reviewPull>>, TError,ReviewPullMutationVariables, TContext> => {

const mutationKey = getReviewPullMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof reviewPull>>, ReviewPullMutationVariables> = (props) => {
          const {owner,repo,number,data} = props ?? {};

          return  reviewPull(owner,repo,number,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type ReviewPullMutationResult = NonNullable<Awaited<ReturnType<typeof reviewPull>>>
    export type ReviewPullMutationBody = ReviewRequest
    export type ReviewPullMutationError = ErrorType<ErrorBody>
    export type ReviewPullMutationVariables = {owner: string;repo: string;number: number;data: ReviewRequest}

    /**
 * Scope: write. Authors can't approve or request changes on their own pull requests.
 * @summary Review a pull request's current head commit
 */
export const useReviewPull = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof reviewPull>>, TError,ReviewPullMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof reviewPull>>,
        TError,
        ReviewPullMutationVariables,
        TContext
      > => {
      return useMutation(getReviewPullMutationOptions(options), queryClient);
    }

export const getGetPullWorkspaceUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/pulls/${encodeURIComponent(String(number))}/view`
}

/**
 * Scope: read.
 * @summary Read a pull request with rendered discussion, merge requirements, and checks
 */
export const getPullWorkspace = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<ItemWorkspace> => {

  return orvalFetch<ItemWorkspace>(getGetPullWorkspaceUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetPullWorkspaceQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/pulls/${number}/view`
    ] as const;
    }


export const getGetPullWorkspaceQueryOptions = <TData = Awaited<ReturnType<typeof getPullWorkspace>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetPullWorkspaceQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getPullWorkspace>>> = ({ signal }) => getPullWorkspace(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getPullWorkspace>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetPullWorkspaceQueryResult = NonNullable<Awaited<ReturnType<typeof getPullWorkspace>>>
export type GetPullWorkspaceQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Read a pull request with rendered discussion, merge requirements, and checks
 */
export function useGetPullWorkspace<TData = Awaited<ReturnType<typeof getPullWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullWorkspace>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPullWorkspace>>,
          TError,
          Awaited<ReturnType<typeof getPullWorkspace>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read a pull request with rendered discussion, merge requirements, and checks
 */
export function useGetPullWorkspace<TData = Awaited<ReturnType<typeof getPullWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullWorkspace>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getPullWorkspace>>,
          TError,
          Awaited<ReturnType<typeof getPullWorkspace>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read a pull request with rendered discussion, merge requirements, and checks
 */
export function useGetPullWorkspace<TData = Awaited<ReturnType<typeof getPullWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Read a pull request with rendered discussion, merge requirements, and checks
 */
export function useGetPullWorkspace<TData = Awaited<ReturnType<typeof getPullWorkspace>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getPullWorkspace>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetPullWorkspaceQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetRepositoryRawFileUrl = (owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/raw?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/raw`
}

/**
 * Scope: read. Served with nosniff and a sandbox policy.
 * @summary Download the original file bytes
 */
export const getRepositoryRawFile = async (owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams, options?: Parameters<typeof orvalFetch>[1]): Promise<Blob | string> => {

  return orvalFetch<Blob | string>(getGetRepositoryRawFileUrl(owner,repo,params),
  {
      responseType: 'blob',
    ...options,
    method: 'GET'


  }
);}





export const getGetRepositoryRawFileQueryKey = (owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams,) => {
    return [
    `/api/repos/${owner}/${repo}/raw`, ...(params ? [params] : [])
    ] as const;
    }


export const getGetRepositoryRawFileQueryOptions = <TData = Awaited<ReturnType<typeof getRepositoryRawFile>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryRawFile>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetRepositoryRawFileQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getRepositoryRawFile>>> = ({ signal }) => getRepositoryRawFile(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getRepositoryRawFile>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetRepositoryRawFileQueryResult = NonNullable<Awaited<ReturnType<typeof getRepositoryRawFile>>>
export type GetRepositoryRawFileQueryError = ErrorType<ErrorBody>


/**
 * Scope: read. Served with nosniff and a sandbox policy.
 * @summary Download the original file bytes
 */
export function useGetRepositoryRawFile<TData = Awaited<ReturnType<typeof getRepositoryRawFile>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  GetRepositoryRawFileParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryRawFile>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepositoryRawFile>>,
          TError,
          Awaited<ReturnType<typeof getRepositoryRawFile>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Served with nosniff and a sandbox policy.
 * @summary Download the original file bytes
 */
export function useGetRepositoryRawFile<TData = Awaited<ReturnType<typeof getRepositoryRawFile>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryRawFile>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepositoryRawFile>>,
          TError,
          Awaited<ReturnType<typeof getRepositoryRawFile>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Served with nosniff and a sandbox policy.
 * @summary Download the original file bytes
 */
export function useGetRepositoryRawFile<TData = Awaited<ReturnType<typeof getRepositoryRawFile>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryRawFile>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Served with nosniff and a sandbox policy.
 * @summary Download the original file bytes
 */
export function useGetRepositoryRawFile<TData = Awaited<ReturnType<typeof getRepositoryRawFile>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryRawFileParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryRawFile>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetRepositoryRawFileQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListReleasesUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases`
}

/**
 * Scope: read. Drafts are listed only with write access.
 * @summary List releases
 */
export const listReleases = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Release[]> => {

  return orvalFetch<Release[]>(getListReleasesUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListReleasesQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/releases`
    ] as const;
    }


export const getListReleasesQueryOptions = <TData = Awaited<ReturnType<typeof listReleases>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listReleases>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListReleasesQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listReleases>>> = ({ signal }) => listReleases(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listReleases>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListReleasesQueryResult = NonNullable<Awaited<ReturnType<typeof listReleases>>>
export type ListReleasesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read. Drafts are listed only with write access.
 * @summary List releases
 */
export function useListReleases<TData = Awaited<ReturnType<typeof listReleases>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listReleases>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listReleases>>,
          TError,
          Awaited<ReturnType<typeof listReleases>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Drafts are listed only with write access.
 * @summary List releases
 */
export function useListReleases<TData = Awaited<ReturnType<typeof listReleases>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listReleases>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listReleases>>,
          TError,
          Awaited<ReturnType<typeof listReleases>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Drafts are listed only with write access.
 * @summary List releases
 */
export function useListReleases<TData = Awaited<ReturnType<typeof listReleases>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listReleases>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Drafts are listed only with write access.
 * @summary List releases
 */
export function useListReleases<TData = Awaited<ReturnType<typeof listReleases>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listReleases>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListReleasesQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateReleaseUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases`
}

/**
 * Scope: write. Publishing creates the tag at target if it doesn't exist; an existing tag is never moved, so target must match it. actions lists the Actions runs it queued.
 * @summary Create a release
 */
export const createRelease = async (owner: string,
    repo: string,
    createReleaseRequest: CreateReleaseRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<ReleaseWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<ReleaseWrite>(getCreateReleaseUrl(owner,repo),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createReleaseRequest)
  }
);}





export const getCreateReleaseMutationKey = () => ['createRelease'] as const;

export const getCreateReleaseMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRelease>>, TError,CreateReleaseMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createRelease>>, TError,CreateReleaseMutationVariables, TContext> => {

const mutationKey = getCreateReleaseMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createRelease>>, CreateReleaseMutationVariables> = (props) => {
          const {owner,repo,data} = props ?? {};

          return  createRelease(owner,repo,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateReleaseMutationResult = NonNullable<Awaited<ReturnType<typeof createRelease>>>
    export type CreateReleaseMutationBody = CreateReleaseRequest
    export type CreateReleaseMutationError = ErrorType<ErrorBody>
    export type CreateReleaseMutationVariables = {owner: string;repo: string;data: CreateReleaseRequest}

    /**
 * Scope: write. Publishing creates the tag at target if it doesn't exist; an existing tag is never moved, so target must match it. actions lists the Actions runs it queued.
 * @summary Create a release
 */
export const useCreateRelease = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRelease>>, TError,CreateReleaseMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createRelease>>,
        TError,
        CreateReleaseMutationVariables,
        TContext
      > => {
      return useMutation(getCreateReleaseMutationOptions(options), queryClient);
    }

export const getGetLatestReleaseUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/latest`
}

/**
 * Scope: read.
 * @summary The most recently published release that isn't a prerelease
 */
export const getLatestRelease = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Release> => {

  return orvalFetch<Release>(getGetLatestReleaseUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetLatestReleaseQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/releases/latest`
    ] as const;
    }


export const getGetLatestReleaseQueryOptions = <TData = Awaited<ReturnType<typeof getLatestRelease>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getLatestRelease>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetLatestReleaseQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getLatestRelease>>> = ({ signal }) => getLatestRelease(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getLatestRelease>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetLatestReleaseQueryResult = NonNullable<Awaited<ReturnType<typeof getLatestRelease>>>
export type GetLatestReleaseQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary The most recently published release that isn't a prerelease
 */
export function useGetLatestRelease<TData = Awaited<ReturnType<typeof getLatestRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getLatestRelease>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getLatestRelease>>,
          TError,
          Awaited<ReturnType<typeof getLatestRelease>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary The most recently published release that isn't a prerelease
 */
export function useGetLatestRelease<TData = Awaited<ReturnType<typeof getLatestRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getLatestRelease>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getLatestRelease>>,
          TError,
          Awaited<ReturnType<typeof getLatestRelease>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary The most recently published release that isn't a prerelease
 */
export function useGetLatestRelease<TData = Awaited<ReturnType<typeof getLatestRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getLatestRelease>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary The most recently published release that isn't a prerelease
 */
export function useGetLatestRelease<TData = Awaited<ReturnType<typeof getLatestRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getLatestRelease>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetLatestReleaseQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getPreviewReleaseNotesUrl = (owner: string,
    repo: string,
    params?: PreviewReleaseNotesParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/notes?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/notes`
}

/**
 * Scope: read.
 * @summary Preview generated release notes
 */
export const previewReleaseNotes = async (owner: string,
    repo: string,
    params?: PreviewReleaseNotesParams, options?: Parameters<typeof orvalFetch>[1]): Promise<ReleaseNotes> => {

  return orvalFetch<ReleaseNotes>(getPreviewReleaseNotesUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getPreviewReleaseNotesMutationKey = () => ['previewReleaseNotes'] as const;

export const getPreviewReleaseNotesMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof previewReleaseNotes>>, TError,PreviewReleaseNotesMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof previewReleaseNotes>>, TError,PreviewReleaseNotesMutationVariables, TContext> => {

const mutationKey = getPreviewReleaseNotesMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof previewReleaseNotes>>, PreviewReleaseNotesMutationVariables> = (props) => {
          const {owner,repo,params} = props ?? {};

          return  previewReleaseNotes(owner,repo,params,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type PreviewReleaseNotesMutationResult = NonNullable<Awaited<ReturnType<typeof previewReleaseNotes>>>

    export type PreviewReleaseNotesMutationError = ErrorType<ErrorBody>
    export type PreviewReleaseNotesMutationVariables = {owner: string;repo: string;params?: PreviewReleaseNotesParams}

    /**
 * Scope: read.
 * @summary Preview generated release notes
 */
export const usePreviewReleaseNotes = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof previewReleaseNotes>>, TError,PreviewReleaseNotesMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof previewReleaseNotes>>,
        TError,
        PreviewReleaseNotesMutationVariables,
        TContext
      > => {
      return useMutation(getPreviewReleaseNotesMutationOptions(options), queryClient);
    }

export const getGetReleaseByTagUrl = (owner: string,
    repo: string,
    tag: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/tags/${encodeURIComponent(String(tag))}`
}

/**
 * Scope: read. The tag may contain slashes.
 * @summary Get a release by tag
 */
export const getReleaseByTag = async (owner: string,
    repo: string,
    tag: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Release> => {

  return orvalFetch<Release>(getGetReleaseByTagUrl(owner,repo,tag),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetReleaseByTagQueryKey = (owner: string,
    repo: string,
    tag: string,) => {
    return [
    `/api/repos/${owner}/${repo}/releases/tags/${tag}`
    ] as const;
    }


export const getGetReleaseByTagQueryOptions = <TData = Awaited<ReturnType<typeof getReleaseByTag>>, TError = ErrorType<unknown>>(owner: string,
    repo: string,
    tag: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getReleaseByTag>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetReleaseByTagQueryKey(owner,repo,tag);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getReleaseByTag>>> = ({ signal }) => getReleaseByTag(owner,repo,tag, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && tag !== null && tag !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getReleaseByTag>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetReleaseByTagQueryResult = NonNullable<Awaited<ReturnType<typeof getReleaseByTag>>>
export type GetReleaseByTagQueryError = ErrorType<unknown>


/**
 * Scope: read. The tag may contain slashes.
 * @summary Get a release by tag
 */
export function useGetReleaseByTag<TData = Awaited<ReturnType<typeof getReleaseByTag>>, TError = ErrorType<unknown>>(
 owner: string,
    repo: string,
    tag: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getReleaseByTag>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getReleaseByTag>>,
          TError,
          Awaited<ReturnType<typeof getReleaseByTag>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. The tag may contain slashes.
 * @summary Get a release by tag
 */
export function useGetReleaseByTag<TData = Awaited<ReturnType<typeof getReleaseByTag>>, TError = ErrorType<unknown>>(
 owner: string,
    repo: string,
    tag: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getReleaseByTag>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getReleaseByTag>>,
          TError,
          Awaited<ReturnType<typeof getReleaseByTag>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. The tag may contain slashes.
 * @summary Get a release by tag
 */
export function useGetReleaseByTag<TData = Awaited<ReturnType<typeof getReleaseByTag>>, TError = ErrorType<unknown>>(
 owner: string,
    repo: string,
    tag: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getReleaseByTag>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. The tag may contain slashes.
 * @summary Get a release by tag
 */
export function useGetReleaseByTag<TData = Awaited<ReturnType<typeof getReleaseByTag>>, TError = ErrorType<unknown>>(
 owner: string,
    repo: string,
    tag: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getReleaseByTag>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetReleaseByTagQueryOptions(owner,repo,tag,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getDeleteReleaseUrl = (owner: string,
    repo: string,
    id: number,
    params?: DeleteReleaseParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}`
}

/**
 * Scope: write.
 * @summary Delete a release and its assets
 */
export const deleteRelease = async (owner: string,
    repo: string,
    id: number,
    params?: DeleteReleaseParams, options?: Parameters<typeof orvalFetch>[1]): Promise<ReleaseDeleted> => {

  return orvalFetch<ReleaseDeleted>(getDeleteReleaseUrl(owner,repo,id,params),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteReleaseMutationKey = () => ['deleteRelease'] as const;

export const getDeleteReleaseMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRelease>>, TError,DeleteReleaseMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteRelease>>, TError,DeleteReleaseMutationVariables, TContext> => {

const mutationKey = getDeleteReleaseMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteRelease>>, DeleteReleaseMutationVariables> = (props) => {
          const {owner,repo,id,params} = props ?? {};

          return  deleteRelease(owner,repo,id,params,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteReleaseMutationResult = NonNullable<Awaited<ReturnType<typeof deleteRelease>>>

    export type DeleteReleaseMutationError = ErrorType<ErrorBody>
    export type DeleteReleaseMutationVariables = {owner: string;repo: string;id: number;params?: DeleteReleaseParams}

    /**
 * Scope: write.
 * @summary Delete a release and its assets
 */
export const useDeleteRelease = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRelease>>, TError,DeleteReleaseMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteRelease>>,
        TError,
        DeleteReleaseMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteReleaseMutationOptions(options), queryClient);
    }

export const getGetReleaseUrl = (owner: string,
    repo: string,
    id: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}`
}

/**
 * Scope: read.
 * @summary Get a release
 */
export const getRelease = async (owner: string,
    repo: string,
    id: number, options?: Parameters<typeof orvalFetch>[1]): Promise<Release> => {

  return orvalFetch<Release>(getGetReleaseUrl(owner,repo,id),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetReleaseQueryKey = (owner: string,
    repo: string,
    id: number,) => {
    return [
    `/api/repos/${owner}/${repo}/releases/${id}`
    ] as const;
    }


export const getGetReleaseQueryOptions = <TData = Awaited<ReturnType<typeof getRelease>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    id: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRelease>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetReleaseQueryKey(owner,repo,id);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getRelease>>> = ({ signal }) => getRelease(owner,repo,id, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && id !== null && id !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getRelease>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetReleaseQueryResult = NonNullable<Awaited<ReturnType<typeof getRelease>>>
export type GetReleaseQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Get a release
 */
export function useGetRelease<TData = Awaited<ReturnType<typeof getRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRelease>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRelease>>,
          TError,
          Awaited<ReturnType<typeof getRelease>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a release
 */
export function useGetRelease<TData = Awaited<ReturnType<typeof getRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRelease>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRelease>>,
          TError,
          Awaited<ReturnType<typeof getRelease>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a release
 */
export function useGetRelease<TData = Awaited<ReturnType<typeof getRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRelease>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a release
 */
export function useGetRelease<TData = Awaited<ReturnType<typeof getRelease>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRelease>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetReleaseQueryOptions(owner,repo,id,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getEditReleaseUrl = (owner: string,
    repo: string,
    id: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}`
}

/**
 * Scope: write. draft: false publishes it.
 * @summary Edit or publish a release
 */
export const editRelease = async (owner: string,
    repo: string,
    id: number,
    editReleaseRequest: EditReleaseRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<ReleaseWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<ReleaseWrite>(getEditReleaseUrl(owner,repo,id),
  {
    ...options,
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(editReleaseRequest)
  }
);}





export const getEditReleaseMutationKey = () => ['editRelease'] as const;

export const getEditReleaseMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof editRelease>>, TError,EditReleaseMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof editRelease>>, TError,EditReleaseMutationVariables, TContext> => {

const mutationKey = getEditReleaseMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof editRelease>>, EditReleaseMutationVariables> = (props) => {
          const {owner,repo,id,data} = props ?? {};

          return  editRelease(owner,repo,id,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type EditReleaseMutationResult = NonNullable<Awaited<ReturnType<typeof editRelease>>>
    export type EditReleaseMutationBody = EditReleaseRequest
    export type EditReleaseMutationError = ErrorType<ErrorBody>
    export type EditReleaseMutationVariables = {owner: string;repo: string;id: number;data: EditReleaseRequest}

    /**
 * Scope: write. draft: false publishes it.
 * @summary Edit or publish a release
 */
export const useEditRelease = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof editRelease>>, TError,EditReleaseMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof editRelease>>,
        TError,
        EditReleaseMutationVariables,
        TContext
      > => {
      return useMutation(getEditReleaseMutationOptions(options), queryClient);
    }

export const getUploadAssetUrl = (owner: string,
    repo: string,
    id: number,
    params: UploadAssetParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}/assets?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}/assets`
}

/**
 * Scope: write. The request body is the file; its Content-Type is stored with it.
 * @summary Upload an asset
 */
export const uploadAsset = async (owner: string,
    repo: string,
    id: number,
    uploadAssetBody: Blob,
    params: UploadAssetParams, options?: Parameters<typeof orvalFetch>[1]): Promise<AssetWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<AssetWrite>(getUploadAssetUrl(owner,repo,id,params),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/octet-stream', ...getHeaders(options?.headers) },
    body: uploadAssetBody
  }
);}





export const getUploadAssetMutationKey = () => ['uploadAsset'] as const;

export const getUploadAssetMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof uploadAsset>>, TError,UploadAssetMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof uploadAsset>>, TError,UploadAssetMutationVariables, TContext> => {

const mutationKey = getUploadAssetMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof uploadAsset>>, UploadAssetMutationVariables> = (props) => {
          const {owner,repo,id,data,params} = props ?? {};

          return  uploadAsset(owner,repo,id,data,params,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type UploadAssetMutationResult = NonNullable<Awaited<ReturnType<typeof uploadAsset>>>
    export type UploadAssetMutationBody = Blob
    export type UploadAssetMutationError = ErrorType<ErrorBody>
    export type UploadAssetMutationVariables = {owner: string;repo: string;id: number;data: Blob;params: UploadAssetParams}

    /**
 * Scope: write. The request body is the file; its Content-Type is stored with it.
 * @summary Upload an asset
 */
export const useUploadAsset = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof uploadAsset>>, TError,UploadAssetMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof uploadAsset>>,
        TError,
        UploadAssetMutationVariables,
        TContext
      > => {
      return useMutation(getUploadAssetMutationOptions(options), queryClient);
    }

export const getDeleteAssetUrl = (owner: string,
    repo: string,
    id: number,
    name: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}/assets/${encodeURIComponent(String(name))}`
}

/**
 * Scope: write.
 * @summary Delete an asset
 */
export const deleteAsset = async (owner: string,
    repo: string,
    id: number,
    name: string, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedWithEvent> => {

  return orvalFetch<DeletedWithEvent>(getDeleteAssetUrl(owner,repo,id,name),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteAssetMutationKey = () => ['deleteAsset'] as const;

export const getDeleteAssetMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteAsset>>, TError,DeleteAssetMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteAsset>>, TError,DeleteAssetMutationVariables, TContext> => {

const mutationKey = getDeleteAssetMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteAsset>>, DeleteAssetMutationVariables> = (props) => {
          const {owner,repo,id,name} = props ?? {};

          return  deleteAsset(owner,repo,id,name,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteAssetMutationResult = NonNullable<Awaited<ReturnType<typeof deleteAsset>>>

    export type DeleteAssetMutationError = ErrorType<ErrorBody>
    export type DeleteAssetMutationVariables = {owner: string;repo: string;id: number;name: string}

    /**
 * Scope: write.
 * @summary Delete an asset
 */
export const useDeleteAsset = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteAsset>>, TError,DeleteAssetMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteAsset>>,
        TError,
        DeleteAssetMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteAssetMutationOptions(options), queryClient);
    }

export const getDownloadAssetUrl = (owner: string,
    repo: string,
    id: number,
    name: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/${encodeURIComponent(String(id))}/assets/${encodeURIComponent(String(name))}`
}

/**
 * Scope: read. Supports range requests.
 * @summary Download an asset
 */
export const downloadAsset = async (owner: string,
    repo: string,
    id: number,
    name: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Blob> => {

  return orvalFetch<Blob>(getDownloadAssetUrl(owner,repo,id,name),
  {
      responseType: 'blob',
    ...options,
    method: 'GET'


  }
);}





export const getDownloadAssetQueryKey = (owner: string,
    repo: string,
    id: number,
    name: string,) => {
    return [
    `/api/repos/${owner}/${repo}/releases/${id}/assets/${name}`
    ] as const;
    }


export const getDownloadAssetQueryOptions = <TData = Awaited<ReturnType<typeof downloadAsset>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    id: number,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAsset>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getDownloadAssetQueryKey(owner,repo,id,name);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof downloadAsset>>> = ({ signal }) => downloadAsset(owner,repo,id,name, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && id !== null && id !== undefined && name !== null && name !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof downloadAsset>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type DownloadAssetQueryResult = NonNullable<Awaited<ReturnType<typeof downloadAsset>>>
export type DownloadAssetQueryError = ErrorType<ErrorBody>


/**
 * Scope: read. Supports range requests.
 * @summary Download an asset
 */
export function useDownloadAsset<TData = Awaited<ReturnType<typeof downloadAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number,
    name: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAsset>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof downloadAsset>>,
          TError,
          Awaited<ReturnType<typeof downloadAsset>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Supports range requests.
 * @summary Download an asset
 */
export function useDownloadAsset<TData = Awaited<ReturnType<typeof downloadAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAsset>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof downloadAsset>>,
          TError,
          Awaited<ReturnType<typeof downloadAsset>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Supports range requests.
 * @summary Download an asset
 */
export function useDownloadAsset<TData = Awaited<ReturnType<typeof downloadAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAsset>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. Supports range requests.
 * @summary Download an asset
 */
export function useDownloadAsset<TData = Awaited<ReturnType<typeof downloadAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    id: number,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAsset>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getDownloadAssetQueryOptions(owner,repo,id,name,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getDeleteRuleUrl = (owner: string,
    repo: string,
    params: DeleteRuleParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/rules?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/rules`
}

/**
 * Scope: admin.
 * @summary Delete a branch rule
 */
export const deleteRule = async (owner: string,
    repo: string,
    params: DeleteRuleParams, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedWithEvent> => {

  return orvalFetch<DeletedWithEvent>(getDeleteRuleUrl(owner,repo,params),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteRuleMutationKey = () => ['deleteRule'] as const;

export const getDeleteRuleMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRule>>, TError,DeleteRuleMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteRule>>, TError,DeleteRuleMutationVariables, TContext> => {

const mutationKey = getDeleteRuleMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteRule>>, DeleteRuleMutationVariables> = (props) => {
          const {owner,repo,params} = props ?? {};

          return  deleteRule(owner,repo,params,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteRuleMutationResult = NonNullable<Awaited<ReturnType<typeof deleteRule>>>

    export type DeleteRuleMutationError = ErrorType<ErrorBody>
    export type DeleteRuleMutationVariables = {owner: string;repo: string;params: DeleteRuleParams}

    /**
 * Scope: admin.
 * @summary Delete a branch rule
 */
export const useDeleteRule = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRule>>, TError,DeleteRuleMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteRule>>,
        TError,
        DeleteRuleMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteRuleMutationOptions(options), queryClient);
    }

export const getListRulesUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/rules`
}

/**
 * Scope: read.
 * @summary List branch rules
 */
export const listRules = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Rule[]> => {

  return orvalFetch<Rule[]>(getListRulesUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListRulesQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/rules`
    ] as const;
    }


export const getListRulesQueryOptions = <TData = Awaited<ReturnType<typeof listRules>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRules>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListRulesQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listRules>>> = ({ signal }) => listRules(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listRules>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListRulesQueryResult = NonNullable<Awaited<ReturnType<typeof listRules>>>
export type ListRulesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List branch rules
 */
export function useListRules<TData = Awaited<ReturnType<typeof listRules>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRules>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRules>>,
          TError,
          Awaited<ReturnType<typeof listRules>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List branch rules
 */
export function useListRules<TData = Awaited<ReturnType<typeof listRules>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRules>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRules>>,
          TError,
          Awaited<ReturnType<typeof listRules>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List branch rules
 */
export function useListRules<TData = Awaited<ReturnType<typeof listRules>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRules>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List branch rules
 */
export function useListRules<TData = Awaited<ReturnType<typeof listRules>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRules>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListRulesQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getSetRuleUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/rules`
}

/**
 * Scope: admin.
 * @summary Create or replace the rule for a branch pattern
 */
export const setRule = async (owner: string,
    repo: string,
    setRuleRequest: SetRuleRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<RuleWrite> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<RuleWrite>(getSetRuleUrl(owner,repo),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(setRuleRequest)
  }
);}





export const getSetRuleMutationKey = () => ['setRule'] as const;

export const getSetRuleMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setRule>>, TError,SetRuleMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof setRule>>, TError,SetRuleMutationVariables, TContext> => {

const mutationKey = getSetRuleMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof setRule>>, SetRuleMutationVariables> = (props) => {
          const {owner,repo,data} = props ?? {};

          return  setRule(owner,repo,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SetRuleMutationResult = NonNullable<Awaited<ReturnType<typeof setRule>>>
    export type SetRuleMutationBody = SetRuleRequest
    export type SetRuleMutationError = ErrorType<ErrorBody>
    export type SetRuleMutationVariables = {owner: string;repo: string;data: SetRuleRequest}

    /**
 * Scope: admin.
 * @summary Create or replace the rule for a branch pattern
 */
export const useSetRule = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setRule>>, TError,SetRuleMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof setRule>>,
        TError,
        SetRuleMutationVariables,
        TContext
      > => {
      return useMutation(getSetRuleMutationOptions(options), queryClient);
    }

export const getListRunsUrl = (owner: string,
    repo: string,
    params?: ListRunsParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs`
}

/**
 * Scope: read.
 * @summary List Actions runs
 */
export const listRuns = async (owner: string,
    repo: string,
    params?: ListRunsParams, options?: Parameters<typeof orvalFetch>[1]): Promise<Run[]> => {

  return orvalFetch<Run[]>(getListRunsUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListRunsQueryKey = (owner: string,
    repo: string,
    params?: ListRunsParams,) => {
    return [
    `/api/repos/${owner}/${repo}/runs`, ...(params ? [params] : [])
    ] as const;
    }


export const getListRunsQueryOptions = <TData = Awaited<ReturnType<typeof listRuns>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: ListRunsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRuns>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListRunsQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listRuns>>> = ({ signal }) => listRuns(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listRuns>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListRunsQueryResult = NonNullable<Awaited<ReturnType<typeof listRuns>>>
export type ListRunsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List Actions runs
 */
export function useListRuns<TData = Awaited<ReturnType<typeof listRuns>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  ListRunsParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRuns>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRuns>>,
          TError,
          Awaited<ReturnType<typeof listRuns>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List Actions runs
 */
export function useListRuns<TData = Awaited<ReturnType<typeof listRuns>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListRunsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRuns>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRuns>>,
          TError,
          Awaited<ReturnType<typeof listRuns>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List Actions runs
 */
export function useListRuns<TData = Awaited<ReturnType<typeof listRuns>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListRunsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRuns>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List Actions runs
 */
export function useListRuns<TData = Awaited<ReturnType<typeof listRuns>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListRunsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRuns>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListRunsQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetRunUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs/${encodeURIComponent(String(number))}`
}

/**
 * Scope: read.
 * @summary Get a run with its jobs and steps
 */
export const getRun = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<Run> => {

  return orvalFetch<Run>(getGetRunUrl(owner,repo,number),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetRunQueryKey = (owner: string,
    repo: string,
    number: number,) => {
    return [
    `/api/repos/${owner}/${repo}/runs/${number}`
    ] as const;
    }


export const getGetRunQueryOptions = <TData = Awaited<ReturnType<typeof getRun>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRun>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetRunQueryKey(owner,repo,number);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getRun>>> = ({ signal }) => getRun(owner,repo,number, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getRun>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetRunQueryResult = NonNullable<Awaited<ReturnType<typeof getRun>>>
export type GetRunQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Get a run with its jobs and steps
 */
export function useGetRun<TData = Awaited<ReturnType<typeof getRun>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRun>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRun>>,
          TError,
          Awaited<ReturnType<typeof getRun>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a run with its jobs and steps
 */
export function useGetRun<TData = Awaited<ReturnType<typeof getRun>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRun>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRun>>,
          TError,
          Awaited<ReturnType<typeof getRun>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a run with its jobs and steps
 */
export function useGetRun<TData = Awaited<ReturnType<typeof getRun>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRun>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Get a run with its jobs and steps
 */
export function useGetRun<TData = Awaited<ReturnType<typeof getRun>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRun>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetRunQueryOptions(owner,repo,number,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCancelRunUrl = (owner: string,
    repo: string,
    number: number,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs/${encodeURIComponent(String(number))}/cancel`
}

/**
 * Scope: write.
 * @summary Cancel a run
 */
export const cancelRun = async (owner: string,
    repo: string,
    number: number, options?: Parameters<typeof orvalFetch>[1]): Promise<RunCancelled> => {

  return orvalFetch<RunCancelled>(getCancelRunUrl(owner,repo,number),
  {
    ...options,
    method: 'POST'


  }
);}





export const getCancelRunMutationKey = () => ['cancelRun'] as const;

export const getCancelRunMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof cancelRun>>, TError,CancelRunMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof cancelRun>>, TError,CancelRunMutationVariables, TContext> => {

const mutationKey = getCancelRunMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof cancelRun>>, CancelRunMutationVariables> = (props) => {
          const {owner,repo,number} = props ?? {};

          return  cancelRun(owner,repo,number,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CancelRunMutationResult = NonNullable<Awaited<ReturnType<typeof cancelRun>>>

    export type CancelRunMutationError = ErrorType<ErrorBody>
    export type CancelRunMutationVariables = {owner: string;repo: string;number: number}

    /**
 * Scope: write.
 * @summary Cancel a run
 */
export const useCancelRun = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof cancelRun>>, TError,CancelRunMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof cancelRun>>,
        TError,
        CancelRunMutationVariables,
        TContext
      > => {
      return useMutation(getCancelRunMutationOptions(options), queryClient);
    }

export const getGetJobLogsUrl = (owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs/${encodeURIComponent(String(number))}/jobs/${encodeURIComponent(String(job))}/logs?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs/${encodeURIComponent(String(number))}/jobs/${encodeURIComponent(String(job))}/logs`
}

/**
 * Scope: read.
 * @summary A job's log lines
 */
export const getJobLogs = async (owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams, options?: Parameters<typeof orvalFetch>[1]): Promise<string> => {

  return orvalFetch<string>(getGetJobLogsUrl(owner,repo,number,job,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetJobLogsQueryKey = (owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams,) => {
    return [
    `/api/repos/${owner}/${repo}/runs/${number}/jobs/${job}/logs`, ...(params ? [params] : [])
    ] as const;
    }


export const getGetJobLogsQueryOptions = <TData = Awaited<ReturnType<typeof getJobLogs>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getJobLogs>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetJobLogsQueryKey(owner,repo,number,job,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getJobLogs>>> = ({ signal }) => getJobLogs(owner,repo,number,job,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && number !== null && number !== undefined && job !== null && job !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getJobLogs>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetJobLogsQueryResult = NonNullable<Awaited<ReturnType<typeof getJobLogs>>>
export type GetJobLogsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary A job's log lines
 */
export function useGetJobLogs<TData = Awaited<ReturnType<typeof getJobLogs>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number,
    job: number,
    params: undefined |  GetJobLogsParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getJobLogs>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getJobLogs>>,
          TError,
          Awaited<ReturnType<typeof getJobLogs>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary A job's log lines
 */
export function useGetJobLogs<TData = Awaited<ReturnType<typeof getJobLogs>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getJobLogs>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getJobLogs>>,
          TError,
          Awaited<ReturnType<typeof getJobLogs>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary A job's log lines
 */
export function useGetJobLogs<TData = Awaited<ReturnType<typeof getJobLogs>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getJobLogs>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary A job's log lines
 */
export function useGetJobLogs<TData = Awaited<ReturnType<typeof getJobLogs>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    number: number,
    job: number,
    params?: GetJobLogsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getJobLogs>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetJobLogsQueryOptions(owner,repo,number,job,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getRerunRunUrl = (owner: string,
    repo: string,
    number: number,
    params?: RerunRunParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs/${encodeURIComponent(String(number))}/rerun?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/runs/${encodeURIComponent(String(number))}/rerun`
}

/**
 * Scope: write.
 * @summary Rerun a run
 */
export const rerunRun = async (owner: string,
    repo: string,
    number: number,
    params?: RerunRunParams, options?: Parameters<typeof orvalFetch>[1]): Promise<RunWrite> => {

  return orvalFetch<RunWrite>(getRerunRunUrl(owner,repo,number,params),
  {
    ...options,
    method: 'POST'


  }
);}





export const getRerunRunMutationKey = () => ['rerunRun'] as const;

export const getRerunRunMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof rerunRun>>, TError,RerunRunMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof rerunRun>>, TError,RerunRunMutationVariables, TContext> => {

const mutationKey = getRerunRunMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof rerunRun>>, RerunRunMutationVariables> = (props) => {
          const {owner,repo,number,params} = props ?? {};

          return  rerunRun(owner,repo,number,params,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type RerunRunMutationResult = NonNullable<Awaited<ReturnType<typeof rerunRun>>>

    export type RerunRunMutationError = ErrorType<ErrorBody>
    export type RerunRunMutationVariables = {owner: string;repo: string;number: number;params?: RerunRunParams}

    /**
 * Scope: write.
 * @summary Rerun a run
 */
export const useRerunRun = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof rerunRun>>, TError,RerunRunMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof rerunRun>>,
        TError,
        RerunRunMutationVariables,
        TContext
      > => {
      return useMutation(getRerunRunMutationOptions(options), queryClient);
    }

export const getListSecretsUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/secrets`
}

/**
 * Scope: admin.
 * @summary List secrets, without their values
 */
export const listSecrets = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Secret[]> => {

  return orvalFetch<Secret[]>(getListSecretsUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListSecretsQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/secrets`
    ] as const;
    }


export const getListSecretsQueryOptions = <TData = Awaited<ReturnType<typeof listSecrets>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listSecrets>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListSecretsQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listSecrets>>> = ({ signal }) => listSecrets(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listSecrets>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListSecretsQueryResult = NonNullable<Awaited<ReturnType<typeof listSecrets>>>
export type ListSecretsQueryError = ErrorType<ErrorBody>


/**
 * Scope: admin.
 * @summary List secrets, without their values
 */
export function useListSecrets<TData = Awaited<ReturnType<typeof listSecrets>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listSecrets>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listSecrets>>,
          TError,
          Awaited<ReturnType<typeof listSecrets>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin.
 * @summary List secrets, without their values
 */
export function useListSecrets<TData = Awaited<ReturnType<typeof listSecrets>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listSecrets>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listSecrets>>,
          TError,
          Awaited<ReturnType<typeof listSecrets>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin.
 * @summary List secrets, without their values
 */
export function useListSecrets<TData = Awaited<ReturnType<typeof listSecrets>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listSecrets>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin.
 * @summary List secrets, without their values
 */
export function useListSecrets<TData = Awaited<ReturnType<typeof listSecrets>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listSecrets>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListSecretsQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getDeleteSecretUrl = (owner: string,
    repo: string,
    name: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/secrets/${encodeURIComponent(String(name))}`
}

/**
 * Scope: admin.
 * @summary Delete a secret
 */
export const deleteSecret = async (owner: string,
    repo: string,
    name: string, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedWithEvent> => {

  return orvalFetch<DeletedWithEvent>(getDeleteSecretUrl(owner,repo,name),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteSecretMutationKey = () => ['deleteSecret'] as const;

export const getDeleteSecretMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteSecret>>, TError,DeleteSecretMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteSecret>>, TError,DeleteSecretMutationVariables, TContext> => {

const mutationKey = getDeleteSecretMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteSecret>>, DeleteSecretMutationVariables> = (props) => {
          const {owner,repo,name} = props ?? {};

          return  deleteSecret(owner,repo,name,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteSecretMutationResult = NonNullable<Awaited<ReturnType<typeof deleteSecret>>>

    export type DeleteSecretMutationError = ErrorType<ErrorBody>
    export type DeleteSecretMutationVariables = {owner: string;repo: string;name: string}

    /**
 * Scope: admin.
 * @summary Delete a secret
 */
export const useDeleteSecret = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteSecret>>, TError,DeleteSecretMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteSecret>>,
        TError,
        DeleteSecretMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteSecretMutationOptions(options), queryClient);
    }

export const getSetSecretUrl = (owner: string,
    repo: string,
    name: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/secrets/${encodeURIComponent(String(name))}`
}

/**
 * Scope: admin.
 * @summary Set a secret
 */
export const setSecret = async (owner: string,
    repo: string,
    name: string,
    valueRequest: ValueRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<SecretSet> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<SecretSet>(getSetSecretUrl(owner,repo,name),
  {
    ...options,
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(valueRequest)
  }
);}





export const getSetSecretMutationKey = () => ['setSecret'] as const;

export const getSetSecretMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setSecret>>, TError,SetSecretMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof setSecret>>, TError,SetSecretMutationVariables, TContext> => {

const mutationKey = getSetSecretMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof setSecret>>, SetSecretMutationVariables> = (props) => {
          const {owner,repo,name,data} = props ?? {};

          return  setSecret(owner,repo,name,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SetSecretMutationResult = NonNullable<Awaited<ReturnType<typeof setSecret>>>
    export type SetSecretMutationBody = ValueRequest
    export type SetSecretMutationError = ErrorType<ErrorBody>
    export type SetSecretMutationVariables = {owner: string;repo: string;name: string;data: ValueRequest}

    /**
 * Scope: admin.
 * @summary Set a secret
 */
export const useSetSecret = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setSecret>>, TError,SetSecretMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof setSecret>>,
        TError,
        SetSecretMutationVariables,
        TContext
      > => {
      return useMutation(getSetSecretMutationOptions(options), queryClient);
    }

export const getListTagsUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/tags`
}

/**
 * Scope: read.
 * @summary List tags
 */
export const listTags = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Tag[]> => {

  return orvalFetch<Tag[]>(getListTagsUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListTagsQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/tags`
    ] as const;
    }


export const getListTagsQueryOptions = <TData = Awaited<ReturnType<typeof listTags>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTags>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListTagsQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listTags>>> = ({ signal }) => listTags(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listTags>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListTagsQueryResult = NonNullable<Awaited<ReturnType<typeof listTags>>>
export type ListTagsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List tags
 */
export function useListTags<TData = Awaited<ReturnType<typeof listTags>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTags>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listTags>>,
          TError,
          Awaited<ReturnType<typeof listTags>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List tags
 */
export function useListTags<TData = Awaited<ReturnType<typeof listTags>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTags>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listTags>>,
          TError,
          Awaited<ReturnType<typeof listTags>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List tags
 */
export function useListTags<TData = Awaited<ReturnType<typeof listTags>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTags>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List tags
 */
export function useListTags<TData = Awaited<ReturnType<typeof listTags>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTags>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListTagsQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getGetRepositoryTreeUrl = (owner: string,
    repo: string,
    params?: GetRepositoryTreeParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/tree?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/tree`
}

/**
 * Scope: read.
 * @summary Browse repository files and rendered README
 */
export const getRepositoryTree = async (owner: string,
    repo: string,
    params?: GetRepositoryTreeParams, options?: Parameters<typeof orvalFetch>[1]): Promise<TreeResponse> => {

  return orvalFetch<TreeResponse>(getGetRepositoryTreeUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetRepositoryTreeQueryKey = (owner: string,
    repo: string,
    params?: GetRepositoryTreeParams,) => {
    return [
    `/api/repos/${owner}/${repo}/tree`, ...(params ? [params] : [])
    ] as const;
    }


export const getGetRepositoryTreeQueryOptions = <TData = Awaited<ReturnType<typeof getRepositoryTree>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: GetRepositoryTreeParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryTree>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetRepositoryTreeQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getRepositoryTree>>> = ({ signal }) => getRepositoryTree(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getRepositoryTree>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetRepositoryTreeQueryResult = NonNullable<Awaited<ReturnType<typeof getRepositoryTree>>>
export type GetRepositoryTreeQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Browse repository files and rendered README
 */
export function useGetRepositoryTree<TData = Awaited<ReturnType<typeof getRepositoryTree>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  GetRepositoryTreeParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryTree>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepositoryTree>>,
          TError,
          Awaited<ReturnType<typeof getRepositoryTree>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Browse repository files and rendered README
 */
export function useGetRepositoryTree<TData = Awaited<ReturnType<typeof getRepositoryTree>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryTreeParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryTree>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getRepositoryTree>>,
          TError,
          Awaited<ReturnType<typeof getRepositoryTree>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Browse repository files and rendered README
 */
export function useGetRepositoryTree<TData = Awaited<ReturnType<typeof getRepositoryTree>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryTreeParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryTree>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Browse repository files and rendered README
 */
export function useGetRepositoryTree<TData = Awaited<ReturnType<typeof getRepositoryTree>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: GetRepositoryTreeParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getRepositoryTree>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetRepositoryTreeQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListVariablesUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/variables`
}

/**
 * Scope: read.
 * @summary List variables
 */
export const listVariables = async (owner: string,
    repo: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Variable[]> => {

  return orvalFetch<Variable[]>(getListVariablesUrl(owner,repo),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListVariablesQueryKey = (owner: string,
    repo: string,) => {
    return [
    `/api/repos/${owner}/${repo}/variables`
    ] as const;
    }


export const getListVariablesQueryOptions = <TData = Awaited<ReturnType<typeof listVariables>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listVariables>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListVariablesQueryKey(owner,repo);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listVariables>>> = ({ signal }) => listVariables(owner,repo, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listVariables>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListVariablesQueryResult = NonNullable<Awaited<ReturnType<typeof listVariables>>>
export type ListVariablesQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary List variables
 */
export function useListVariables<TData = Awaited<ReturnType<typeof listVariables>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listVariables>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listVariables>>,
          TError,
          Awaited<ReturnType<typeof listVariables>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List variables
 */
export function useListVariables<TData = Awaited<ReturnType<typeof listVariables>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listVariables>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listVariables>>,
          TError,
          Awaited<ReturnType<typeof listVariables>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List variables
 */
export function useListVariables<TData = Awaited<ReturnType<typeof listVariables>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listVariables>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary List variables
 */
export function useListVariables<TData = Awaited<ReturnType<typeof listVariables>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listVariables>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListVariablesQueryOptions(owner,repo,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getDeleteVariableUrl = (owner: string,
    repo: string,
    name: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/variables/${encodeURIComponent(String(name))}`
}

/**
 * Scope: write.
 * @summary Delete a variable
 */
export const deleteVariable = async (owner: string,
    repo: string,
    name: string, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedWithEvent> => {

  return orvalFetch<DeletedWithEvent>(getDeleteVariableUrl(owner,repo,name),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteVariableMutationKey = () => ['deleteVariable'] as const;

export const getDeleteVariableMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteVariable>>, TError,DeleteVariableMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteVariable>>, TError,DeleteVariableMutationVariables, TContext> => {

const mutationKey = getDeleteVariableMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteVariable>>, DeleteVariableMutationVariables> = (props) => {
          const {owner,repo,name} = props ?? {};

          return  deleteVariable(owner,repo,name,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteVariableMutationResult = NonNullable<Awaited<ReturnType<typeof deleteVariable>>>

    export type DeleteVariableMutationError = ErrorType<ErrorBody>
    export type DeleteVariableMutationVariables = {owner: string;repo: string;name: string}

    /**
 * Scope: write.
 * @summary Delete a variable
 */
export const useDeleteVariable = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteVariable>>, TError,DeleteVariableMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteVariable>>,
        TError,
        DeleteVariableMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteVariableMutationOptions(options), queryClient);
    }

export const getSetVariableUrl = (owner: string,
    repo: string,
    name: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/variables/${encodeURIComponent(String(name))}`
}

/**
 * Scope: write.
 * @summary Set a variable
 */
export const setVariable = async (owner: string,
    repo: string,
    name: string,
    valueRequest: ValueRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<VariableSet> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<VariableSet>(getSetVariableUrl(owner,repo,name),
  {
    ...options,
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(valueRequest)
  }
);}





export const getSetVariableMutationKey = () => ['setVariable'] as const;

export const getSetVariableMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setVariable>>, TError,SetVariableMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof setVariable>>, TError,SetVariableMutationVariables, TContext> => {

const mutationKey = getSetVariableMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof setVariable>>, SetVariableMutationVariables> = (props) => {
          const {owner,repo,name,data} = props ?? {};

          return  setVariable(owner,repo,name,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type SetVariableMutationResult = NonNullable<Awaited<ReturnType<typeof setVariable>>>
    export type SetVariableMutationBody = ValueRequest
    export type SetVariableMutationError = ErrorType<ErrorBody>
    export type SetVariableMutationVariables = {owner: string;repo: string;name: string;data: ValueRequest}

    /**
 * Scope: write.
 * @summary Set a variable
 */
export const useSetVariable = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof setVariable>>, TError,SetVariableMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof setVariable>>,
        TError,
        SetVariableMutationVariables,
        TContext
      > => {
      return useMutation(getSetVariableMutationOptions(options), queryClient);
    }

export const getListWorkflowsUrl = (owner: string,
    repo: string,
    params?: ListWorkflowsParams,) => {
  const normalizedParams = new URLSearchParams();

  Object.entries(params || {}).forEach(([key, value]) => {

    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : String(value))
    }
  });

  const stringifiedParams = normalizedParams.toString();

  return stringifiedParams.length > 0 ? `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/workflows?${stringifiedParams}` : `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/workflows`
}

/**
 * Scope: read.
 * @summary Discover workflows and declared manual inputs at a ref
 */
export const listWorkflows = async (owner: string,
    repo: string,
    params?: ListWorkflowsParams, options?: Parameters<typeof orvalFetch>[1]): Promise<WorkflowWorkspace[]> => {

  return orvalFetch<WorkflowWorkspace[]>(getListWorkflowsUrl(owner,repo,params),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListWorkflowsQueryKey = (owner: string,
    repo: string,
    params?: ListWorkflowsParams,) => {
    return [
    `/api/repos/${owner}/${repo}/workflows`, ...(params ? [params] : [])
    ] as const;
    }


export const getListWorkflowsQueryOptions = <TData = Awaited<ReturnType<typeof listWorkflows>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    params?: ListWorkflowsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listWorkflows>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListWorkflowsQueryKey(owner,repo,params);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listWorkflows>>> = ({ signal }) => listWorkflows(owner,repo,params, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listWorkflows>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListWorkflowsQueryResult = NonNullable<Awaited<ReturnType<typeof listWorkflows>>>
export type ListWorkflowsQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Discover workflows and declared manual inputs at a ref
 */
export function useListWorkflows<TData = Awaited<ReturnType<typeof listWorkflows>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params: undefined |  ListWorkflowsParams, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listWorkflows>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listWorkflows>>,
          TError,
          Awaited<ReturnType<typeof listWorkflows>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Discover workflows and declared manual inputs at a ref
 */
export function useListWorkflows<TData = Awaited<ReturnType<typeof listWorkflows>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListWorkflowsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listWorkflows>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listWorkflows>>,
          TError,
          Awaited<ReturnType<typeof listWorkflows>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Discover workflows and declared manual inputs at a ref
 */
export function useListWorkflows<TData = Awaited<ReturnType<typeof listWorkflows>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListWorkflowsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listWorkflows>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Discover workflows and declared manual inputs at a ref
 */
export function useListWorkflows<TData = Awaited<ReturnType<typeof listWorkflows>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    params?: ListWorkflowsParams, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listWorkflows>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListWorkflowsQueryOptions(owner,repo,params,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getDispatchWorkflowUrl = (owner: string,
    repo: string,) => {




  return `/api/repos/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/workflows/dispatch`
}

/**
 * Scope: write. Unknown inputs are refused, and choice and boolean values are checked.
 * @summary Run a workflow with workflow_dispatch
 */
export const dispatchWorkflow = async (owner: string,
    repo: string,
    dispatchRequest: DispatchRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<Run> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<Run>(getDispatchWorkflowUrl(owner,repo),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(dispatchRequest)
  }
);}





export const getDispatchWorkflowMutationKey = () => ['dispatchWorkflow'] as const;

export const getDispatchWorkflowMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof dispatchWorkflow>>, TError,DispatchWorkflowMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof dispatchWorkflow>>, TError,DispatchWorkflowMutationVariables, TContext> => {

const mutationKey = getDispatchWorkflowMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof dispatchWorkflow>>, DispatchWorkflowMutationVariables> = (props) => {
          const {owner,repo,data} = props ?? {};

          return  dispatchWorkflow(owner,repo,data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DispatchWorkflowMutationResult = NonNullable<Awaited<ReturnType<typeof dispatchWorkflow>>>
    export type DispatchWorkflowMutationBody = DispatchRequest
    export type DispatchWorkflowMutationError = ErrorType<ErrorBody>
    export type DispatchWorkflowMutationVariables = {owner: string;repo: string;data: DispatchRequest}

    /**
 * Scope: write. Unknown inputs are refused, and choice and boolean values are checked.
 * @summary Run a workflow with workflow_dispatch
 */
export const useDispatchWorkflow = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof dispatchWorkflow>>, TError,DispatchWorkflowMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof dispatchWorkflow>>,
        TError,
        DispatchWorkflowMutationVariables,
        TContext
      > => {
      return useMutation(getDispatchWorkflowMutationOptions(options), queryClient);
    }

export const getListRunnersUrl = () => {




  return `/api/runners`
}

/**
 * Scope: admin, unrestricted. online means seen in the last minute.
 * @summary List runners
 */
export const listRunners = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<RunnerView[]> => {

  return orvalFetch<RunnerView[]>(getListRunnersUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListRunnersQueryKey = () => {
    return [
    `/api/runners`
    ] as const;
    }


export const getListRunnersQueryOptions = <TData = Awaited<ReturnType<typeof listRunners>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunners>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListRunnersQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listRunners>>> = ({ signal }) => listRunners({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listRunners>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListRunnersQueryResult = NonNullable<Awaited<ReturnType<typeof listRunners>>>
export type ListRunnersQueryError = ErrorType<ErrorBody>


/**
 * Scope: admin, unrestricted. online means seen in the last minute.
 * @summary List runners
 */
export function useListRunners<TData = Awaited<ReturnType<typeof listRunners>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunners>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRunners>>,
          TError,
          Awaited<ReturnType<typeof listRunners>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted. online means seen in the last minute.
 * @summary List runners
 */
export function useListRunners<TData = Awaited<ReturnType<typeof listRunners>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunners>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRunners>>,
          TError,
          Awaited<ReturnType<typeof listRunners>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted. online means seen in the last minute.
 * @summary List runners
 */
export function useListRunners<TData = Awaited<ReturnType<typeof listRunners>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunners>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted. online means seen in the last minute.
 * @summary List runners
 */
export function useListRunners<TData = Awaited<ReturnType<typeof listRunners>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunners>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListRunnersQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateRunnerUrl = () => {




  return `/api/runners`
}

/**
 * Scope: admin, unrestricted. The token is returned once.
 * @summary Create a runner for a config-file connection
 */
export const createRunner = async (nameRequest: NameRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<RunnerCreated> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<RunnerCreated>(getCreateRunnerUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(nameRequest)
  }
);}





export const getCreateRunnerMutationKey = () => ['createRunner'] as const;

export const getCreateRunnerMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRunner>>, TError,CreateRunnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createRunner>>, TError,CreateRunnerMutationVariables, TContext> => {

const mutationKey = getCreateRunnerMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createRunner>>, CreateRunnerMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  createRunner(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateRunnerMutationResult = NonNullable<Awaited<ReturnType<typeof createRunner>>>
    export type CreateRunnerMutationBody = NameRequest
    export type CreateRunnerMutationError = ErrorType<ErrorBody>
    export type CreateRunnerMutationVariables = {data: NameRequest}

    /**
 * Scope: admin, unrestricted. The token is returned once.
 * @summary Create a runner for a config-file connection
 */
export const useCreateRunner = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRunner>>, TError,CreateRunnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createRunner>>,
        TError,
        CreateRunnerMutationVariables,
        TContext
      > => {
      return useMutation(getCreateRunnerMutationOptions(options), queryClient);
    }

export const getListRunnerTokensUrl = () => {




  return `/api/runners/tokens`
}

/**
 * Scope: admin, unrestricted.
 * @summary List registration tokens
 */
export const listRunnerTokens = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<RunnerToken[]> => {

  return orvalFetch<RunnerToken[]>(getListRunnerTokensUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListRunnerTokensQueryKey = () => {
    return [
    `/api/runners/tokens`
    ] as const;
    }


export const getListRunnerTokensQueryOptions = <TData = Awaited<ReturnType<typeof listRunnerTokens>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunnerTokens>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListRunnerTokensQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listRunnerTokens>>> = ({ signal }) => listRunnerTokens({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listRunnerTokens>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListRunnerTokensQueryResult = NonNullable<Awaited<ReturnType<typeof listRunnerTokens>>>
export type ListRunnerTokensQueryError = ErrorType<ErrorBody>


/**
 * Scope: admin, unrestricted.
 * @summary List registration tokens
 */
export function useListRunnerTokens<TData = Awaited<ReturnType<typeof listRunnerTokens>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunnerTokens>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRunnerTokens>>,
          TError,
          Awaited<ReturnType<typeof listRunnerTokens>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary List registration tokens
 */
export function useListRunnerTokens<TData = Awaited<ReturnType<typeof listRunnerTokens>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunnerTokens>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listRunnerTokens>>,
          TError,
          Awaited<ReturnType<typeof listRunnerTokens>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary List registration tokens
 */
export function useListRunnerTokens<TData = Awaited<ReturnType<typeof listRunnerTokens>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunnerTokens>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary List registration tokens
 */
export function useListRunnerTokens<TData = Awaited<ReturnType<typeof listRunnerTokens>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listRunnerTokens>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListRunnerTokensQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateRunnerTokenUrl = () => {




  return `/api/runners/tokens`
}

/**
 * Scope: admin, unrestricted. The secret is returned once.
 * @summary Create a registration token for forgejo-runner register
 */
export const createRunnerToken = async (nameRequest: NameRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<RunnerTokenCreated> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<RunnerTokenCreated>(getCreateRunnerTokenUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(nameRequest)
  }
);}





export const getCreateRunnerTokenMutationKey = () => ['createRunnerToken'] as const;

export const getCreateRunnerTokenMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRunnerToken>>, TError,CreateRunnerTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createRunnerToken>>, TError,CreateRunnerTokenMutationVariables, TContext> => {

const mutationKey = getCreateRunnerTokenMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createRunnerToken>>, CreateRunnerTokenMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  createRunnerToken(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateRunnerTokenMutationResult = NonNullable<Awaited<ReturnType<typeof createRunnerToken>>>
    export type CreateRunnerTokenMutationBody = NameRequest
    export type CreateRunnerTokenMutationError = ErrorType<ErrorBody>
    export type CreateRunnerTokenMutationVariables = {data: NameRequest}

    /**
 * Scope: admin, unrestricted. The secret is returned once.
 * @summary Create a registration token for forgejo-runner register
 */
export const useCreateRunnerToken = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createRunnerToken>>, TError,CreateRunnerTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createRunnerToken>>,
        TError,
        CreateRunnerTokenMutationVariables,
        TContext
      > => {
      return useMutation(getCreateRunnerTokenMutationOptions(options), queryClient);
    }

export const getDeleteRunnerTokenUrl = (id: number,) => {




  return `/api/runners/tokens/${encodeURIComponent(String(id))}`
}

/**
 * Scope: admin, unrestricted. Runners it registered are kept.
 * @summary Delete a registration token
 */
export const deleteRunnerToken = async (id: number, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedIDWithEvent> => {

  return orvalFetch<DeletedIDWithEvent>(getDeleteRunnerTokenUrl(id),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteRunnerTokenMutationKey = () => ['deleteRunnerToken'] as const;

export const getDeleteRunnerTokenMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRunnerToken>>, TError,DeleteRunnerTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteRunnerToken>>, TError,DeleteRunnerTokenMutationVariables, TContext> => {

const mutationKey = getDeleteRunnerTokenMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteRunnerToken>>, DeleteRunnerTokenMutationVariables> = (props) => {
          const {id} = props ?? {};

          return  deleteRunnerToken(id,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteRunnerTokenMutationResult = NonNullable<Awaited<ReturnType<typeof deleteRunnerToken>>>

    export type DeleteRunnerTokenMutationError = ErrorType<ErrorBody>
    export type DeleteRunnerTokenMutationVariables = {id: number}

    /**
 * Scope: admin, unrestricted. Runners it registered are kept.
 * @summary Delete a registration token
 */
export const useDeleteRunnerToken = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRunnerToken>>, TError,DeleteRunnerTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteRunnerToken>>,
        TError,
        DeleteRunnerTokenMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteRunnerTokenMutationOptions(options), queryClient);
    }

export const getDeleteRunnerUrl = (id: number,) => {




  return `/api/runners/${encodeURIComponent(String(id))}`
}

/**
 * Scope: admin, unrestricted.
 * @summary Delete a runner
 */
export const deleteRunner = async (id: number, options?: Parameters<typeof orvalFetch>[1]): Promise<DeletedIDWithEvent> => {

  return orvalFetch<DeletedIDWithEvent>(getDeleteRunnerUrl(id),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getDeleteRunnerMutationKey = () => ['deleteRunner'] as const;

export const getDeleteRunnerMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRunner>>, TError,DeleteRunnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof deleteRunner>>, TError,DeleteRunnerMutationVariables, TContext> => {

const mutationKey = getDeleteRunnerMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof deleteRunner>>, DeleteRunnerMutationVariables> = (props) => {
          const {id} = props ?? {};

          return  deleteRunner(id,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type DeleteRunnerMutationResult = NonNullable<Awaited<ReturnType<typeof deleteRunner>>>

    export type DeleteRunnerMutationError = ErrorType<ErrorBody>
    export type DeleteRunnerMutationVariables = {id: number}

    /**
 * Scope: admin, unrestricted.
 * @summary Delete a runner
 */
export const useDeleteRunner = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof deleteRunner>>, TError,DeleteRunnerMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof deleteRunner>>,
        TError,
        DeleteRunnerMutationVariables,
        TContext
      > => {
      return useMutation(getDeleteRunnerMutationOptions(options), queryClient);
    }

export const getGetSigningKeyUrl = () => {




  return `/api/signing-key`
}

/**
 * Scope: any. Includes an allowed_signers line for git verify-commit.
 * @summary Lyra's own commit signing key
 */
export const getSigningKey = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<SigningKeyView> => {

  return orvalFetch<SigningKeyView>(getGetSigningKeyUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getGetSigningKeyQueryKey = () => {
    return [
    `/api/signing-key`
    ] as const;
    }


export const getGetSigningKeyQueryOptions = <TData = Awaited<ReturnType<typeof getSigningKey>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getSigningKey>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getGetSigningKeyQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof getSigningKey>>> = ({ signal }) => getSigningKey({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof getSigningKey>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type GetSigningKeyQueryResult = NonNullable<Awaited<ReturnType<typeof getSigningKey>>>
export type GetSigningKeyQueryError = ErrorType<ErrorBody>


/**
 * Scope: any. Includes an allowed_signers line for git verify-commit.
 * @summary Lyra's own commit signing key
 */
export function useGetSigningKey<TData = Awaited<ReturnType<typeof getSigningKey>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof getSigningKey>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof getSigningKey>>,
          TError,
          Awaited<ReturnType<typeof getSigningKey>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any. Includes an allowed_signers line for git verify-commit.
 * @summary Lyra's own commit signing key
 */
export function useGetSigningKey<TData = Awaited<ReturnType<typeof getSigningKey>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getSigningKey>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof getSigningKey>>,
          TError,
          Awaited<ReturnType<typeof getSigningKey>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any. Includes an allowed_signers line for git verify-commit.
 * @summary Lyra's own commit signing key
 */
export function useGetSigningKey<TData = Awaited<ReturnType<typeof getSigningKey>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getSigningKey>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: any. Includes an allowed_signers line for git verify-commit.
 * @summary Lyra's own commit signing key
 */
export function useGetSigningKey<TData = Awaited<ReturnType<typeof getSigningKey>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof getSigningKey>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getGetSigningKeyQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getListTokensUrl = () => {




  return `/api/tokens`
}

/**
 * Scope: admin, unrestricted.
 * @summary List the caller's tokens
 */
export const listTokens = async ( options?: Parameters<typeof orvalFetch>[1]): Promise<Token[]> => {

  return orvalFetch<Token[]>(getListTokensUrl(),
  {
    ...options,
    method: 'GET'


  }
);}





export const getListTokensQueryKey = () => {
    return [
    `/api/tokens`
    ] as const;
    }


export const getListTokensQueryOptions = <TData = Awaited<ReturnType<typeof listTokens>>, TError = ErrorType<ErrorBody>>( options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTokens>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getListTokensQueryKey();



    const queryFn: QueryFunction<Awaited<ReturnType<typeof listTokens>>> = ({ signal }) => listTokens({ signal, ...requestOptions });





   return  { queryKey, queryFn, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof listTokens>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type ListTokensQueryResult = NonNullable<Awaited<ReturnType<typeof listTokens>>>
export type ListTokensQueryError = ErrorType<ErrorBody>


/**
 * Scope: admin, unrestricted.
 * @summary List the caller's tokens
 */
export function useListTokens<TData = Awaited<ReturnType<typeof listTokens>>, TError = ErrorType<ErrorBody>>(
  options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTokens>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof listTokens>>,
          TError,
          Awaited<ReturnType<typeof listTokens>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary List the caller's tokens
 */
export function useListTokens<TData = Awaited<ReturnType<typeof listTokens>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTokens>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof listTokens>>,
          TError,
          Awaited<ReturnType<typeof listTokens>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary List the caller's tokens
 */
export function useListTokens<TData = Awaited<ReturnType<typeof listTokens>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTokens>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: admin, unrestricted.
 * @summary List the caller's tokens
 */
export function useListTokens<TData = Awaited<ReturnType<typeof listTokens>>, TError = ErrorType<ErrorBody>>(
  options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof listTokens>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getListTokensQueryOptions(options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getCreateTokenUrl = () => {




  return `/api/tokens`
}

/**
 * Scope: admin, unrestricted. The secret is returned once; only its SHA-256 hash is stored.
 * @summary Create a token
 */
export const createToken = async (createTokenRequest: CreateTokenRequest, options?: Parameters<typeof orvalFetch>[1]): Promise<TokenCreated> => {

    const getHeaders = (h?: NonNullable<RequestInit['headers']>): Record<string, string | readonly string[]> => {
    if (!h) return {};
    if (h instanceof Headers) return Object.fromEntries(h.entries());
    if (Symbol.iterator in h) {
      return Object.fromEntries(
        Array.from(h as Iterable<Iterable<string>>, (entry) => Array.from(entry) as [string, string]),
      );
    }
    const headers: Record<string, string | readonly string[]> = {};
    for (const [name, value] of Object.entries<string | readonly string[] | undefined>(h)) {
      if (value !== undefined) headers[name] = value;
    }
    return headers;
  };
return orvalFetch<TokenCreated>(getCreateTokenUrl(),
  {
    ...options,
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getHeaders(options?.headers) },
    body: JSON.stringify(createTokenRequest)
  }
);}





export const getCreateTokenMutationKey = () => ['createToken'] as const;

export const getCreateTokenMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createToken>>, TError,CreateTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof createToken>>, TError,CreateTokenMutationVariables, TContext> => {

const mutationKey = getCreateTokenMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof createToken>>, CreateTokenMutationVariables> = (props) => {
          const {data} = props ?? {};

          return  createToken(data,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type CreateTokenMutationResult = NonNullable<Awaited<ReturnType<typeof createToken>>>
    export type CreateTokenMutationBody = CreateTokenRequest
    export type CreateTokenMutationError = ErrorType<ErrorBody>
    export type CreateTokenMutationVariables = {data: CreateTokenRequest}

    /**
 * Scope: admin, unrestricted. The secret is returned once; only its SHA-256 hash is stored.
 * @summary Create a token
 */
export const useCreateToken = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof createToken>>, TError,CreateTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof createToken>>,
        TError,
        CreateTokenMutationVariables,
        TContext
      > => {
      return useMutation(getCreateTokenMutationOptions(options), queryClient);
    }

export const getRevokeTokenUrl = (id: number,) => {




  return `/api/tokens/${encodeURIComponent(String(id))}`
}

/**
 * Scope: admin, unrestricted.
 * @summary Revoke a token
 */
export const revokeToken = async (id: number, options?: Parameters<typeof orvalFetch>[1]): Promise<RevokedID> => {

  return orvalFetch<RevokedID>(getRevokeTokenUrl(id),
  {
    ...options,
    method: 'DELETE'


  }
);}





export const getRevokeTokenMutationKey = () => ['revokeToken'] as const;

export const getRevokeTokenMutationOptions = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof revokeToken>>, TError,RevokeTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
): UseMutationOptions<Awaited<ReturnType<typeof revokeToken>>, TError,RevokeTokenMutationVariables, TContext> => {

const mutationKey = getRevokeTokenMutationKey();
const {mutation: mutationOptions, request: requestOptions} = options ?
      options.mutation && 'mutationKey' in options.mutation && options.mutation.mutationKey ?
      options
      : {...options, mutation: {...options.mutation, mutationKey}}
      : {mutation: { mutationKey, }, request: undefined};




      const mutationFn: MutationFunction<Awaited<ReturnType<typeof revokeToken>>, RevokeTokenMutationVariables> = (props) => {
          const {id} = props ?? {};

          return  revokeToken(id,requestOptions)
        }






  return  { mutationFn, ...mutationOptions }}

    export type RevokeTokenMutationResult = NonNullable<Awaited<ReturnType<typeof revokeToken>>>

    export type RevokeTokenMutationError = ErrorType<ErrorBody>
    export type RevokeTokenMutationVariables = {id: number}

    /**
 * Scope: admin, unrestricted.
 * @summary Revoke a token
 */
export const useRevokeToken = <TError = ErrorType<ErrorBody>,
    TContext = unknown>(options?: { mutation?:UseMutationOptions<Awaited<ReturnType<typeof revokeToken>>, TError,RevokeTokenMutationVariables, TContext>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient): UseMutationResult<
        Awaited<ReturnType<typeof revokeToken>>,
        TError,
        RevokeTokenMutationVariables,
        TContext
      > => {
      return useMutation(getRevokeTokenMutationOptions(options), queryClient);
    }

export const getDownloadAssetByTagUrl = (owner: string,
    repo: string,
    tag: string,
    name: string,) => {




  return `/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/download/${encodeURIComponent(String(tag))}/${encodeURIComponent(String(name))}`
}

/**
 * Scope: read. A stable URL with range support; the tag may contain slashes.
 * @summary Download an asset by tag
 */
export const downloadAssetByTag = async (owner: string,
    repo: string,
    tag: string,
    name: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Blob> => {

  return orvalFetch<Blob>(getDownloadAssetByTagUrl(owner,repo,tag,name),
  {
      responseType: 'blob',
    ...options,
    method: 'GET'


  }
);}





export const getDownloadAssetByTagQueryKey = (owner: string,
    repo: string,
    tag: string,
    name: string,) => {
    return [
    `/${owner}/${repo}/releases/download/${tag}/${name}`
    ] as const;
    }


export const getDownloadAssetByTagQueryOptions = <TData = Awaited<ReturnType<typeof downloadAssetByTag>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    tag: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAssetByTag>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getDownloadAssetByTagQueryKey(owner,repo,tag,name);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof downloadAssetByTag>>> = ({ signal }) => downloadAssetByTag(owner,repo,tag,name, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && tag !== null && tag !== undefined && name !== null && name !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof downloadAssetByTag>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type DownloadAssetByTagQueryResult = NonNullable<Awaited<ReturnType<typeof downloadAssetByTag>>>
export type DownloadAssetByTagQueryError = ErrorType<ErrorBody>


/**
 * Scope: read. A stable URL with range support; the tag may contain slashes.
 * @summary Download an asset by tag
 */
export function useDownloadAssetByTag<TData = Awaited<ReturnType<typeof downloadAssetByTag>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    tag: string,
    name: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAssetByTag>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof downloadAssetByTag>>,
          TError,
          Awaited<ReturnType<typeof downloadAssetByTag>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. A stable URL with range support; the tag may contain slashes.
 * @summary Download an asset by tag
 */
export function useDownloadAssetByTag<TData = Awaited<ReturnType<typeof downloadAssetByTag>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    tag: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAssetByTag>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof downloadAssetByTag>>,
          TError,
          Awaited<ReturnType<typeof downloadAssetByTag>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. A stable URL with range support; the tag may contain slashes.
 * @summary Download an asset by tag
 */
export function useDownloadAssetByTag<TData = Awaited<ReturnType<typeof downloadAssetByTag>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    tag: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAssetByTag>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read. A stable URL with range support; the tag may contain slashes.
 * @summary Download an asset by tag
 */
export function useDownloadAssetByTag<TData = Awaited<ReturnType<typeof downloadAssetByTag>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    tag: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadAssetByTag>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getDownloadAssetByTagQueryOptions(owner,repo,tag,name,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







export const getDownloadLatestAssetUrl = (owner: string,
    repo: string,
    name: string,) => {




  return `/${encodeURIComponent(String(owner))}/${encodeURIComponent(String(repo))}/releases/latest/download/${encodeURIComponent(String(name))}`
}

/**
 * Scope: read.
 * @summary Download an asset of the latest release
 */
export const downloadLatestAsset = async (owner: string,
    repo: string,
    name: string, options?: Parameters<typeof orvalFetch>[1]): Promise<Blob> => {

  return orvalFetch<Blob>(getDownloadLatestAssetUrl(owner,repo,name),
  {
      responseType: 'blob',
    ...options,
    method: 'GET'


  }
);}





export const getDownloadLatestAssetQueryKey = (owner: string,
    repo: string,
    name: string,) => {
    return [
    `/${owner}/${repo}/releases/latest/download/${name}`
    ] as const;
    }


export const getDownloadLatestAssetQueryOptions = <TData = Awaited<ReturnType<typeof downloadLatestAsset>>, TError = ErrorType<ErrorBody>>(owner: string,
    repo: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadLatestAsset>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
) => {

const {query: queryOptions, request: requestOptions} = options ?? {};

  const queryKey =  queryOptions?.queryKey ?? getDownloadLatestAssetQueryKey(owner,repo,name);



    const queryFn: QueryFunction<Awaited<ReturnType<typeof downloadLatestAsset>>> = ({ signal }) => downloadLatestAsset(owner,repo,name, { signal, ...requestOptions });





   return  { queryKey, queryFn, enabled: owner !== null && owner !== undefined && repo !== null && repo !== undefined && name !== null && name !== undefined, ...queryOptions} as UseQueryOptions<Awaited<ReturnType<typeof downloadLatestAsset>>, TError, TData> & { queryKey: DataTag<QueryKey, TData, TError> }
}

export type DownloadLatestAssetQueryResult = NonNullable<Awaited<ReturnType<typeof downloadLatestAsset>>>
export type DownloadLatestAssetQueryError = ErrorType<ErrorBody>


/**
 * Scope: read.
 * @summary Download an asset of the latest release
 */
export function useDownloadLatestAsset<TData = Awaited<ReturnType<typeof downloadLatestAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    name: string, options: { query:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadLatestAsset>>, TError, TData>> & Pick<
        DefinedInitialDataOptions<
          Awaited<ReturnType<typeof downloadLatestAsset>>,
          TError,
          Awaited<ReturnType<typeof downloadLatestAsset>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  DefinedUseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Download an asset of the latest release
 */
export function useDownloadLatestAsset<TData = Awaited<ReturnType<typeof downloadLatestAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadLatestAsset>>, TError, TData>> & Pick<
        UndefinedInitialDataOptions<
          Awaited<ReturnType<typeof downloadLatestAsset>>,
          TError,
          Awaited<ReturnType<typeof downloadLatestAsset>>
        > , 'initialData'
      >, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Download an asset of the latest release
 */
export function useDownloadLatestAsset<TData = Awaited<ReturnType<typeof downloadLatestAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadLatestAsset>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
  ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> }
/**
 * Scope: read.
 * @summary Download an asset of the latest release
 */
export function useDownloadLatestAsset<TData = Awaited<ReturnType<typeof downloadLatestAsset>>, TError = ErrorType<ErrorBody>>(
 owner: string,
    repo: string,
    name: string, options?: { query?:Partial<UseQueryOptions<Awaited<ReturnType<typeof downloadLatestAsset>>, TError, TData>>, request?: SecondParameter<typeof orvalFetch>}
 , queryClient?: QueryClient
 ):  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {

  const queryOptions = getDownloadLatestAssetQueryOptions(owner,repo,name,options)

  const query = useQuery(queryOptions, queryClient) as  UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> };

  return withQueryKey(query, queryOptions.queryKey);
}







