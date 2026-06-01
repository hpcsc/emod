import { describe, it, expect } from 'vitest';
import { Export } from '../static/emod-export.js';

describe('Export', () => {
  describe('exportToEmodString', () => {
    it('exports model name only with trailing newline', () => {
      var result = Export.exportToEmodString({ modelName: 'Test', nodes: [], edges: [] });
      expect(result).toBe('model "Test"\n');
    });

    it('exports actors after model name with blank line', () => {
      var store = {
        modelName: 'Hotel',
        nodes: [
          { id: 'a1', type: 'actor', label: 'Guest' },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);
      expect(result).toBe('model "Hotel"\n\nactor "Guest"\n');
    });

    it('exports multiple actors with blank lines between', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'a1', type: 'actor', label: 'Guest' },
          { id: 'a2', type: 'actor', label: 'Admin' },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);
      expect(result).toBe('model "Test"\n\nactor "Guest"\n\nactor "Admin"\n');
    });

    it('exports context with aggregate and slice but no children', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Orders' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Order' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Create Order' },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Orders" {',
        '  aggregate "Order" {',
        '    slice "Create Order" {',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports command with fields column-aligned', () => {
      var store = {
        modelName: 'Hotel',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Reservations' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Reservation' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Make Reservation' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'MakeReservation',
            fields: [
              { name: 'guestId', type: 'string', modifier: 'required' },
              { name: 'roomType', type: 'string', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Hotel"',
        '',
        'context "Reservations" {',
        '  aggregate "Reservation" {',
        '    slice "Make Reservation" {',
        '      command MakeReservation {',
        '        fields {',
        '          guestId  string required',
        '          roomType string required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports command and event with blank line between', () => {
      var store = {
        modelName: 'Hotel',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Reservations' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Reservation' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Make Reservation' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'MakeReservation',
            fields: [
              { name: 'guestId', type: 'string', modifier: 'required' },
              { name: 'roomType', type: 'string', modifier: 'required' },
            ],
          },
          {
            id: 'evt1', type: 'event', parentId: 'sl1', label: 'ReservationMade',
            fields: [
              { name: 'reservationId', type: 'string', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Hotel"',
        '',
        'context "Reservations" {',
        '  aggregate "Reservation" {',
        '    slice "Make Reservation" {',
        '      command MakeReservation {',
        '        fields {',
        '          guestId  string required',
        '          roomType string required',
        '        }',
        '      }',
        '',
        '      event ReservationMade {',
        '        fields {',
        '          reservationId string required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports trigger with kind, actor, and reads', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'My Slice' },
          {
            id: 'trg1', type: 'trigger', parentId: 'sl1', label: 'Form',
            kind: 'UI', actor: 'Guest', reads: 'MyView',
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "My Slice" {',
        '      trigger UI "Form" {',
        '        actor Guest',
        '        reads MyView',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports view with fields and subscribes', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'My Slice' },
          {
            id: 'view1', type: 'view', parentId: 'sl1', label: 'RoomsView',
            fields: [
              { name: 'roomId', type: 'string', modifier: 'required' },
              { name: 'status', type: 'string', modifier: 'optional' },
            ],
            subscribes: ['RoomReserved', 'GuestCheckedOut'],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "My Slice" {',
        '      view RoomsView {',
        '        fields {',
        '          roomId string required',
        '          status string optional',
        '        }',
        '        subscribes [RoomReserved, GuestCheckedOut]',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports automation with trigger_event, command, target_context', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Notify' },
          {
            id: 'auto1', type: 'automation', parentId: 'sl1', label: 'OrderNotifier',
            trigger_event: 'OrderPlaced',
            command: 'SendNotification',
            target_context: 'Notifications',
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "Notify" {',
        '      automation OrderNotifier {',
        '        trigger OrderPlaced',
        '        command SendNotification',
        '        target context Notifications',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports translation with external_system, reads, command', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Import' },
          {
            id: 'trans1', type: 'translation', parentId: 'sl1', label: 'BookingImport',
            external_system: 'Booking.com API',
            reads: 'WebhookView',
            command: 'ImportBooking',
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "Import" {',
        '      translation BookingImport {',
        '        external_system "Booking.com API"',
        '        reads WebhookView',
        '        command ImportBooking',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports translation with inline event and field alignment', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Import' },
          {
            id: 'trans1', type: 'translation', parentId: 'sl1', label: 'BookingImport',
            external_system: 'Booking.com API',
            reads: 'WebhookView',
            command: 'ImportBooking',
            event: {
              name: 'BookingImported',
              fields: [
                { name: 'bookingId', type: 'string', modifier: 'required' },
                { name: 'source', type: 'string', modifier: 'required' },
              ],
            },
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "Import" {',
        '      translation BookingImport {',
        '        external_system "Booking.com API"',
        '        reads WebhookView',
        '        command ImportBooking',
        '        event BookingImported {',
        '          fields {',
        '            bookingId string required',
        '            source    string required',
        '          }',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports event with source external', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Receive' },
          {
            id: 'evt1', type: 'event', parentId: 'sl1', label: 'PaymentReceived',
            source: 'external',
            external_name: 'Stripe',
            fields: [
              { name: 'paymentId', type: 'string', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "Receive" {',
        '      event PaymentReceived {',
        '        source external "Stripe"',
        '        fields {',
        '          paymentId string required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports event with no fields omits fields block', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'evt1', type: 'event', parentId: 'sl1', label: 'ThingHappened',
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      event ThingHappened {',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports view with subscribes but no fields omits fields block', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'view1', type: 'view', parentId: 'sl1', label: 'MyView',
            subscribes: ['EventA'],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      view MyView {',
        '        subscribes [EventA]',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports view with fields but no subscribes omits subscribes line', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'view1', type: 'view', parentId: 'sl1', label: 'MyView',
            fields: [
              { name: 'id', type: 'string' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      view MyView {',
        '        fields {',
        '          id string',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports flow edges referencing labels from edges where type is flow', () => {
      var store = {
        modelName: 'Hotel',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Reservations' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Reservation' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Make Reservation' },
          { id: 'cmd1', type: 'command', parentId: 'sl1', label: 'MakeReservation' },
          { id: 'evt1', type: 'event', parentId: 'sl1', label: 'ReservationMade' },
        ],
        edges: [
          { source: 'cmd1', target: 'evt1', type: 'flow' },
        ],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Hotel"',
        '',
        'context "Reservations" {',
        '  aggregate "Reservation" {',
        '    slice "Make Reservation" {',
        '      command MakeReservation {',
        '      }',
        '',
        '      event ReservationMade {',
        '      }',
        '',
        '      flow {',
        '        command -> event: MakeReservation -> ReservationMade',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports fields without modifier omit trailing whitespace', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'Cmd',
            fields: [
              { name: 'name', type: 'string' },
              { name: 'age', type: 'int', modifier: 'optional' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      command Cmd {',
        '        fields {',
        '          name string',
        '          age  int    optional',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('field names padded to longest name width within a block', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'Cmd',
            fields: [
              { name: 'id', type: 'string', modifier: 'required' },
              { name: 'guestName', type: 'string', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      command Cmd {',
        '        fields {',
        '          id        string required',
        '          guestName string required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('field types padded to longest type width within a block', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'evt1', type: 'event', parentId: 'sl1', label: 'Evt',
            fields: [
              { name: 'checkIn', type: 'date', modifier: 'required' },
              { name: 'created', type: 'timestamp', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      event Evt {',
        '        fields {',
        '          checkIn date      required',
        '          created timestamp required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('different fields blocks are aligned independently', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'Cmd',
            fields: [
              { name: 'id', type: 'string', modifier: 'required' },
              { name: 'name', type: 'string', modifier: 'required' },
            ],
          },
          {
            id: 'evt1', type: 'event', parentId: 'sl1', label: 'Evt',
            fields: [
              { name: 'reservationId', type: 'string', modifier: 'required' },
              { name: 'at', type: 'timestamp', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      command Cmd {',
        '        fields {',
        '          id   string required',
        '          name string required',
        '        }',
        '      }',
        '',
        '      event Evt {',
        '        fields {',
        '          reservationId string    required',
        '          at            timestamp required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('single-field block produces no extra padding', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'Cmd',
            fields: [
              { name: 'id', type: 'string', modifier: 'required' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      command Cmd {',
        '        fields {',
        '          id string required',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports all node types in canonical order', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'Full Slice' },
          {
            id: 'trg1', type: 'trigger', parentId: 'sl1', label: 'Form',
            kind: 'UI', actor: 'User',
          },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'DoThing',
          },
          {
            id: 'evt1', type: 'event', parentId: 'sl1', label: 'ThingDone',
          },
          {
            id: 'view1', type: 'view', parentId: 'sl1', label: 'ThingView',
            subscribes: ['ThingDone'],
          },
          {
            id: 'auto1', type: 'automation', parentId: 'sl1', label: 'Reactor',
            trigger_event: 'ThingDone',
            command: 'Notify',
          },
        ],
        edges: [
          { source: 'cmd1', target: 'evt1', type: 'flow' },
        ],
      };
      var result = Export.exportToEmodString(store);

      // Verify order of elements within slice
      var triggerIdx = result.indexOf('trigger UI');
      var commandIdx = result.indexOf('command DoThing');
      var eventIdx = result.indexOf('event ThingDone');
      var viewIdx = result.indexOf('view ThingView');
      var autoIdx = result.indexOf('automation Reactor');
      var flowIdx = result.indexOf('flow {');

      expect(triggerIdx).toBeGreaterThan(-1);
      expect(commandIdx).toBeGreaterThan(triggerIdx);
      expect(eventIdx).toBeGreaterThan(commandIdx);
      expect(viewIdx).toBeGreaterThan(eventIdx);
      expect(autoIdx).toBeGreaterThan(viewIdx);
      expect(flowIdx).toBeGreaterThan(autoIdx);
    });

    it('exports blank line between slices', () => {
      var store = {
        modelName: 'Hotel',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'First' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'CmdA',
          },
          { id: 'sl2', type: 'slice', parentId: 'agg1', label: 'Second' },
          {
            id: 'cmd2', type: 'command', parentId: 'sl2', label: 'CmdB',
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Hotel"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "First" {',
        '      command CmdA {',
        '      }',
        '    }',
        '',
        '    slice "Second" {',
        '      command CmdB {',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports multiple contexts with blank lines between', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Orders' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Order' },
          { id: 'ctx2', type: 'context', label: 'Payments' },
          { id: 'agg2', type: 'aggregate', parentId: 'ctx2', label: 'Payment' },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Orders" {',
        '  aggregate "Order" {',
        '  }',
        '}',
        '',
        'context "Payments" {',
        '  aggregate "Payment" {',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });

    it('exports actor node type as actor %q format', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'a1', type: 'actor', label: 'Guest' },
          { id: 'a2', type: 'actor', label: 'Admin' },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);
      expect(result).toContain('actor "Guest"');
      expect(result).toContain('actor "Admin"');
    });

    it('exports mixed modifier and no-modifier fields aligned correctly', () => {
      var store = {
        modelName: 'Test',
        nodes: [
          { id: 'ctx1', type: 'context', label: 'Ctx' },
          { id: 'agg1', type: 'aggregate', parentId: 'ctx1', label: 'Agg' },
          { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S' },
          {
            id: 'cmd1', type: 'command', parentId: 'sl1', label: 'Cmd',
            fields: [
              { name: 'firstName', type: 'string', modifier: 'required' },
              { name: 'age', type: 'int' },
              { name: 'email', type: 'string', modifier: 'optional' },
            ],
          },
        ],
        edges: [],
      };
      var result = Export.exportToEmodString(store);

      var expected = [
        'model "Test"',
        '',
        'context "Ctx" {',
        '  aggregate "Agg" {',
        '    slice "S" {',
        '      command Cmd {',
        '        fields {',
        '          firstName string required',
        '          age       int',
        '          email     string optional',
        '        }',
        '      }',
        '    }',
        '  }',
        '}',
        '',
      ].join('\n');

      expect(result).toBe(expected);
    });
  });
});
