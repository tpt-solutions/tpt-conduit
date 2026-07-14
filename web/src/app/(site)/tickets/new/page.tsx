import { graphqlFetch } from "@/lib/graphql";
import { WORKFLOWS_QUERY } from "@/lib/queries";
import { Workflow } from "@/lib/types";
import { TicketForm } from "@/components/TicketForm";

export default async function NewTicketPage() {
  const data = await graphqlFetch<{ workflows: Workflow[] }>(WORKFLOWS_QUERY);

  return (
    <>
      <div className="page-header">
        <h1>New ticket</h1>
      </div>
      <TicketForm workflows={data.workflows} />
    </>
  );
}
