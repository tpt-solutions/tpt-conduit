export const TICKETS_QUERY = /* GraphQL */ `
  {
    tickets {
      id
      workflow
      workflowVersion
      title
      fields
      assignee
      queue
      priority
      createdAt
      updatedAt
    }
  }
`;

export const TICKET_QUERY = /* GraphQL */ `
  query ($id: String!) {
    ticket(id: $id) {
      id
      workflow
      workflowVersion
      title
      fields
      assignee
      queue
      priority
      createdAt
      updatedAt
    }
  }
`;

export const WORKFLOWS_QUERY = /* GraphQL */ `
  {
    workflows {
      name
      version
      description
      initial
      steps {
        name
        kind
        task
        next
        onError
        assignTo
        approval {
          chain {
            role
            user
          }
        }
      }
    }
  }
`;

export const RUNS_QUERY = /* GraphQL */ `
  {
    runs {
      id
      ticketId
      workflow
      workflowVersion
      status
      currentStep
      failed
      createdAt
      updatedAt
    }
  }
`;

export const RUN_QUERY = /* GraphQL */ `
  query ($id: String!) {
    run(id: $id) {
      id
      ticketId
      workflow
      workflowVersion
      status
      currentStep
      output
      failed
      createdAt
      updatedAt
      steps {
        name
        kind
        status
        attempt
        output
        error
        dueAt
        approval {
          chain {
            role
            user
          }
          index
          status
          decidedBy
        }
      }
    }
  }
`;

export const EVENTS_QUERY = /* GraphQL */ `
  query ($runId: String!) {
    events(runId: $runId) {
      seq
      type
      at
      payload
      scheduleAt
    }
  }
`;

export const CREATE_TICKET_MUTATION = /* GraphQL */ `
  mutation ($input: CreateTicketInput!) {
    createTicket(input: $input) {
      id
      title
      workflow
      workflowVersion
    }
  }
`;

export const APPROVE_MUTATION = /* GraphQL */ `
  mutation ($runId: String!, $step: String!, $by: String!, $comment: String) {
    approve(runId: $runId, step: $step, by: $by, comment: $comment)
  }
`;

export const REJECT_MUTATION = /* GraphQL */ `
  mutation ($runId: String!, $step: String!, $by: String!, $reason: String) {
    reject(runId: $runId, step: $step, by: $by, reason: $reason)
  }
`;

export const CANCEL_MUTATION = /* GraphQL */ `
  mutation ($runId: String!, $reason: String) {
    cancel(runId: $runId, reason: $reason)
  }
`;
