/**
 * @link https://prisma.io/docs/support/help-articles/nextjs-prisma-client-dev-practices
 */
import { type Prisma, PrismaClient } from "@prisma/client";
import { env } from "@/env.mjs";
import { omit } from "lodash";

type DBContext = {
  userEmail: string
  orgId: string
};



export interface ContextAwarePrismaClient extends PrismaClient {


  /**
   *  This returns a client is which is essentially a proxy or wrapper around the original Prisma client;
   *  it ensures that the specified context is associated with any subsequent database queries made using this client.
   */
  $setContext: (context: DBContext) => ContextAwarePrismaClient;
}


const globalForPrisma = global as unknown as {
  prisma: ContextAwarePrismaClient | undefined
}

export const prisma=  globalForPrisma.prisma ||  createClient();


// This proxy intercepts property access
// and modifies function calls to include the context if the property is a function
function addContextProxy(model: object, _context: DBContext) {
  return new Proxy(model, {
    get(target, p, receiver) {
      const method = Reflect.get(target, p, receiver);

      if (typeof method !== 'function'  || p === 'transaction') {
        return method as unknown;
      }

      return (args: unknown) => {
        if (typeof args !== 'object') {
          return method.call(target, args) as unknown;
        }

        return method.call(target, { ...args, _context }) as unknown; // modify function with context
      };
    },
  });
}

/**
 *  used to set the database context for the Prisma client.
 * @returns a proxy that wraps the Prisma client, ensuring that any queries made with it include the context information.
 * @param _context
 */
export function setContext(this: ContextAwarePrismaClient, _context: DBContext): ContextAwarePrismaClient {

  console.log(`received request`)
  return new Proxy(this, {
    get(target, p, receiver) {
      const original = Reflect.get(target, p, receiver) as unknown | object;

      if (typeof p !== 'string' || /^\$.+/.test(p) || typeof original !== 'object'
      ) {
        return original;
      }

      return addContextProxy(original as object, _context);
    },
  });
}


/**
 *
 * used to prevent context from being passed to another middleware
 */
const consumeContextMiddleware: Prisma.Middleware = async (params, next) => {

  return await next(omit(params, 'args._context')) as unknown;
};


 function createClient(): ContextAwarePrismaClient {

   // wrap prisma client with our custom method
  const temp = Object.create(new PrismaClient({ log: env.PRISMA_LOG_QUERIES ? ['query'] : []}), {
    $setContext: {
      value: setContext,
      enumerable: false,
      writable: false,
      configurable: false,
    },
  }) as ContextAwarePrismaClient;


  console.log("Node Env: ", env.NODE_ENV)

   return temp

  // const cacheMiddleware: Prisma.Middleware = createPrismaRedisCache({
  //     storage: {
  //         type: "redis",
  //         options: {
  //             client: redis,
  //             invalidation: {
  //                 referencesTTL: 120
  //             },
  //             log: console,
  //         }
  //     },
  //
  //     // storage: {
  //     //     type: "redis",
  //     //     options: {
  //     //         client: redis,
  //     //         invalidation: {
  //     //             referencesTTL: 120
  //     //         },
  //     //         log: console
  //     //     }
  //     // },
  //     cacheTime: 300,
  //     excludeModels: ["Submission", "GradingEvents"],
  //     excludeMethods: ["count", "groupBy"],
  //     onHit: (key) => {
  //         console.log("hit", key);
  //     },
  //     onMiss: (key) => {
  //         console.log("miss", key);
  //     },
  //     onError: (key) => {
  //         console.log("error", key);
  //     },
  // });

  // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
  // temp.$use(cacheMiddleware)

}



prisma.$use(async (params, next) => {
  let event = params.action as string;

  //Create, Update Course
  if (params.model === "Course" && (params.action === "create" || params.action === "update")) {
    const context: DBContext = params.args._context;

    if (context && context.userEmail) {
      try {

        const user = await prisma.user.findFirstOrThrow({
          where: {
            email: context.userEmail
          }
        })

        console.log(`Prisma middleware user id ${user.id}`)

        const course = await prisma.course.findFirstOrThrow({
          where: {
            id: params.args.data.id,
          },
        });

        if (params.action === "create"){
          event = "create"
        }
        else if (params.action === "update"){
          event = "update"
        }


        console.log(`context: ${JSON.stringify(context)}`)
        await prisma.auditLog.create({
          data: {
            orgId: context.orgId,
            userId: user.id,
            eventName: event,
            description: `User ${context.userEmail} ${event}d course ${course.name}`,
            timeStamp: new Date(),
            ObjectId: course.id,
          },
        });
        console.log(`created audit log`)
      } catch (error) {
        console.log(`error creating audit log ${JSON.stringify(error)}`)
      }
    }
  }

  //COURSE TEMPLATES

  //Meant to be triggered on : createCourseTemplate and removeCourseTemplate
  if (params.model === "CourseTemplate" && (params.action === "create" || params.action === "update")) {
    const context: DBContext = params.args._context;

    if (context && context.userEmail) {
      try {

        const user = await prisma.user.findFirstOrThrow({
          where: {
            email: context.userEmail
          }
        })

        console.log(`Prisma middleware user id ${user.id}`)

        const courseTemplate = await prisma.courseTemplate.findFirstOrThrow({
          where: {
            id: params.args.data.id,
          },
        });

        if (params.action === "create") event = "create";
        else if (params.action === "update"){
          if(courseTemplate?.isDeleted == false && params.args.data.isDeleted == true) {
            event = "delete"
          }
          else return next(params);
        }


        console.log(`context: ${JSON.stringify(context)}`)
        await prisma.auditLog.create({
          data: {
            orgId: context.orgId,
            userId: user.id,
            eventName: event,
            description: `User ${context.userEmail} ${event}d course template ${courseTemplate.name}`,
            timeStamp: new Date(),
            ObjectId: courseTemplate.id,
          },
        });
        console.log(`created audit log`)
      } catch (error) {
        console.log(`error creating audit log ${JSON.stringify(error)}`)
      }
    }
  }

  //ASSIGNMENT TEMPLATES

  //Meant to be triggered on : createTemplate
  if (params.model === "AssignmentTemplate" && params.action === "create") {
    const context: DBContext = params.args._context;

    if (context && context.userEmail) {
      try {

        const user = await prisma.user.findFirstOrThrow({
          where: {
            email: context.userEmail
          }
        })

        console.log(`Prisma middleware user id ${user.id}`)

        const createdAssignmentTemplate = await prisma.assignmentTemplate.findFirstOrThrow({
          where: {
            id: params.args.data.id,
          },
        });


        console.log(`context: ${JSON.stringify(context)}`)
        await prisma.auditLog.create({
          data: {
            orgId: context.orgId,
            userId: user.id,
            eventName: "Create",
            description: `User ${context.userEmail} created assignment template ${createdAssignmentTemplate.name}`,
            timeStamp: new Date(),
            ObjectId: createdAssignmentTemplate.id,
          },
        });
        console.log(`created audit log`)
      } catch (error) {
        console.log(`error creating audit log ${JSON.stringify(error)}`)
      }
    }
  }

  //Meant to be triggered on : removeAssignmentTemplateFromCourseTemplate and addAssignmentTemplatetoCourseTemplate
  if (params.model === "CourseTemplateAssignmentTemplate" && (params.action === "delete" || params.action === "create")) {
    const context: DBContext = params.args._context;
    let assignment;
    let course;

    if (context && context.userEmail) {
      try {

        const user = await prisma.user.findFirstOrThrow({
          where: {
            email: context.userEmail
          }
        })

        if(params.action === "delete") {
          event = "delete"
          assignment = await prisma.assignmentTemplate.findFirstOrThrow({
            where: {
              id: params.args.where.courseTemplateId_assignmentTemplateId.assignmentTemplateId,
            }
          })
          course = await prisma.courseTemplate.findFirstOrThrow({
            where: {
              id: params.args.where.courseTemplateId_assignmentTemplateId.courseTemplateId,
            }
          })
        }

        else{
          event = "create"
          assignment = await prisma.assignmentTemplate.findFirstOrThrow({
            where: {
              id: params.args.data.assignmentTemplateId,
            }
          })

          course = await prisma.courseTemplate.findFirstOrThrow({
            where: {
              id: params.args.data.courseTemplateId,
            }
          })
        }

        console.log(`Prisma middleware user id ${user.id}`)

        console.log(`context: ${JSON.stringify(context)}`)
        await prisma.auditLog.create({
          data: {
            orgId: context.orgId,
            userId: user.id,
            eventName: event,
            description: `User ${context.userEmail} ${event}d assignment template ${assignment.name} in ${course.name}`,
            timeStamp: new Date(),
            ObjectId: assignment.id,
          },
        });
        console.log(`created audit log`)
      } catch (error) {
        console.log(`error creating audit log ${JSON.stringify(error)}`)
      }
    }
  }

  //ASSIGNMENTS

  //Meant to be triggered on : publishAssignment, unpublishAssignment, editAssignment, convertAssignmentTemplateToAssignment
  if (params.model === "Assignment" && (params.action === "update" || params.action === "create")) {
    const context: DBContext = params.args._context;

    if (context && context.userEmail) {
      try {

        const user = await prisma.user.findFirstOrThrow({
          where: {
            email: context.userEmail
          }
        })

      let assignment;

      if(params.action === "update") {
        assignment = await prisma.assignment.findFirstOrThrow({
          where: {
            id: params.args.data.id,
          }
        })

        if (assignment?.published == false && params.args.data.published == true) {
          event = "published";
        }
        else if (assignment?.published == true && params.args.data.published == false) {
          event = "unpublished";
        }
        else {
          event = "edited";
        }
      }

      else{
        event = "created"
        assignment = await prisma.assignmentTemplate.findFirstOrThrow({
          where: {
            id: params.args.data.id,
          }
        })
      }

        console.log(`Prisma middleware user id ${user.id}`)

        console.log(`context: ${JSON.stringify(context)}`)
        await prisma.auditLog.create({
          data: {
            orgId: context.orgId,
            userId: user.id,
            eventName: event,
            description: `User ${context.userEmail} ${event} assignment ${assignment.name}`,
            timeStamp: new Date(),
            ObjectId: assignment.id,
          },
        });
        console.log(`created audit log`)
      } catch (error) {
        console.log(`error creating audit log ${JSON.stringify(error)}`)
      }
    }
  }

  if (params.model === "SubmissionVersion" && params.action === "create") {
    const context: DBContext = params.args._context;

    if (context && context.userEmail && params.args.data.adhoc) {
      try {

        const user = await prisma.user.findFirstOrThrow({
          where: {
            email: context.userEmail
          }
        })

        console.log(`Prisma middleware user id ${user.id}`)

        const submissionVersion = await prisma.submissionVersion.findFirstOrThrow({
          where: {
            id: params.args.data.id,
          },
        });


        console.log(`context: ${JSON.stringify(context)}`)
        await prisma.auditLog.create({
          data: {
            orgId: context.orgId,
            userId: user.id,
            eventName: "Regrade",
            description: `User ${context.userEmail} regraded submission ${submissionVersion.submissionId}`,
            timeStamp: new Date(),
            ObjectId: submissionVersion.submissionId,
          },
        });
        console.log(`created audit log`)
      } catch (error) {
        console.log(`error creating audit log ${JSON.stringify(error)}`)
      }
    }
  }


  return next(params)
})

prisma.$use(consumeContextMiddleware)


if (process.env.NODE_ENV !== 'production') {
    globalForPrisma.prisma = prisma
}
