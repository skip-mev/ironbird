import axios from 'axios';
import type { TestnetWorkflowRequest, WorkflowResponse, WorkflowStatus, LoadTestSpec } from '../types/workflow';

const API_BASE_URL = '/ironbird';

export const workflowApi = {
  createWorkflow: async (request: TestnetWorkflowRequest): Promise<WorkflowResponse> => {
    const response = await axios.post(`${API_BASE_URL}/workflow`, request);
    return response.data;
  },

  listWorkflows: async (): Promise<{Workflows: Array<{WorkflowID: string; Status: string; StartTime: string; Repo?: string; SHA?: string}>; Count: number}> => {
    const response = await axios.get(`${API_BASE_URL}/workflows`);
    return response.data;
  },

  updateWorkflow: async (workflowId: string, request: TestnetWorkflowRequest): Promise<WorkflowResponse> => {
    const response = await axios.put(`${API_BASE_URL}/workflow/${workflowId}`, request);
    return response.data;
  },

  getWorkflow: async (workflowId: string): Promise<WorkflowStatus> => {
    const response = await axios.get(`${API_BASE_URL}/workflow/${workflowId}`);
    return response.data;
  },

  runLoadTest: async (workflowId: string, spec: LoadTestSpec): Promise<WorkflowResponse> => {
    const response = await axios.post(`${API_BASE_URL}/loadtest/${workflowId}`, spec);
    return response.data;
  },

  cancelWorkflow: async (workflowId: string): Promise<WorkflowResponse> => {
    const response = await axios.post(`${API_BASE_URL}/workflow/${workflowId}/cancel`);
    return response.data;
  },

  sendShutdownSignal: async (workflowId: string): Promise<WorkflowResponse> => {
    const response = await axios.post(`${API_BASE_URL}/workflow/${workflowId}/signal/shutdown`);
    return response.data;
  },
};
