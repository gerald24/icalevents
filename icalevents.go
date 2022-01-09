package icalevents

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework EventKit -framework Foundation

#import <string.h>
#import <stdlib.h>
#import <EventKit/EventKit.h>
#import <Foundation/Foundation.h>

EKEventStore* eventStore;
NSCalendar* gregorian;
NSDateFormatter* df;
unsigned unitFlags = NSCalendarUnitYear | NSCalendarUnitMonth | NSCalendarUnitDay;

typedef struct CEvent {
	char* title;
	char* location;
	char* notes;
	char* startDate;
	char* endDate;
	char* duration;
} CEvent;

typedef struct CEventsResult {
	char* error;
	int count;
	CEvent** events;
} CEventsResult;

CEvent* event_at(CEvent** events, int idx) {
	return events[idx];
}

NSString* initEventStore() {
	if (eventStore != nil) {
		return NULL;
	}

	NSLog(@"init event store");

	dispatch_semaphore_t mySemaphore = dispatch_semaphore_create(0);
	__block BOOL success;

	EKEventStore* es = [[EKEventStore alloc] init];
	[es requestAccessToEntityType:EKEntityTypeEvent completion:^(BOOL granted, NSError *error) {
		success = granted;
		dispatch_semaphore_signal(mySemaphore);
	}];

	dispatch_semaphore_wait(mySemaphore, DISPATCH_TIME_FOREVER);

	if (!success) {
		return @"calendar not accessible";
	}

	NSLog(@"event store access granted.");
	eventStore = es;
	gregorian = [[NSCalendar alloc] initWithCalendarIdentifier: NSCalendarIdentifierGregorian];
	df = [[NSDateFormatter alloc] init];
	[df setCalendar: gregorian];
	[df setDateFormat:@"yyyy-MM-dd HH:mm"];

	return NULL;
}

CEventsResult* createErrorResult(NSString* errorMessage) {
	CEventsResult* result = malloc(sizeof(struct CEventsResult *));
	result->error = strdup((char*)[errorMessage UTF8String]);
	result->count = 0;
	return result;
}

CEventsResult* LoadCalendarNamed(char* calendarCName) {
	@autoreleasepool {
		NSString* err = initEventStore();
		if (err != NULL) {
			return createErrorResult(err);
		}

		NSString* calendarName = [NSString stringWithCString:calendarCName encoding:NSUTF8StringEncoding];

		EKCalendar* calendar = nil;
		for (EKCalendar* cal in [eventStore calendarsForEntityType: EKEntityTypeEvent]) {
			if ([cal.title isEqual: calendarName])
				calendar = cal;
		}
		if (calendar == nil) {
			return createErrorResult(@"calendar not found");
		}

		// currently recent 2 years - may pass from-to as parameter someday
		// Remember eventStore predicateForEventsWithStartDate:endDate:calendars: will restrict event to 4 years !
		NSDateComponents* dateComponents = [gregorian components: unitFlags fromDate: [NSDate date]];
		[dateComponents setDay: 31];
		[dateComponents setHour: 23];
		[dateComponents setMinute: 59];
		[dateComponents setSecond: 59];
		NSDate* endDate = [gregorian dateFromComponents: dateComponents];

		[dateComponents setYear: (dateComponents.year - 2)];
		[dateComponents setDay: 1];
		[dateComponents setHour: 0];
		[dateComponents setMinute: 0];
		[dateComponents setSecond: 0];
		NSDate* startDate = [gregorian dateFromComponents: dateComponents];

		NSPredicate* predicate = [eventStore predicateForEventsWithStartDate: startDate endDate: endDate calendars: [NSArray arrayWithObject: calendar]];

		NSArray* unsortedEvents = [eventStore eventsMatchingPredicate: predicate];

		NSMutableArray* sortedEvents = [unsortedEvents mutableCopy];
		NSSortDescriptor* desc = [[NSSortDescriptor alloc] initWithKey: @"startDate" ascending: NO];
		[sortedEvents sortUsingDescriptors: [NSArray arrayWithObject: desc]];

		CEventsResult* result = malloc(sizeof(struct CEventsResult));
		result->count = [sortedEvents count];
		if (result->count > 0) {
			result->events = malloc(result->count * sizeof(struct CEvent *));
			int idx = 0;
			for (EKEvent* event in sortedEvents) {
				CEvent* cevent = malloc(sizeof(struct CEvent));
				cevent->title = strdup([(event.title == nil ? @"" : event.title) UTF8String]);
				cevent->location = strdup([(event.location == nil ? @"" : event.location) UTF8String]);
				cevent->notes = strdup([(event.notes == nil ? @"" : event.notes) UTF8String]);
				cevent->startDate = strdup([[df stringFromDate: event.startDate] UTF8String]);
				cevent->endDate = strdup([[df stringFromDate: event.endDate] UTF8String]);
				cevent->duration = strdup([[NSString stringWithFormat:@"%lf", [event.endDate timeIntervalSinceDate: event.startDate] / 60] UTF8String]);
				result->events[idx++] = cevent;
			}
		}

		return result;
	}
}

*/
import "C"
import (
	"errors"
	"time"
	"unsafe"
)

type Event struct {
	Title     string
	Location  string
	Notes     string
	StartDate time.Time
	EndDate   time.Time
	Duration  time.Duration
}

func Events(calendar string) ([]Event, error) {
	var events []Event

	calstr := C.CString(calendar)
	result := C.LoadCalendarNamed(calstr)
	count := int(result.count)
	defer func() {
		C.free(unsafe.Pointer(calstr))

		if count > 0 {
			cevents := (**C.CEvent)(unsafe.Pointer(result.events))
			for i := 0; i < count; i++ {
				ptr := C.event_at(cevents, C.int(i))
				cevent := (*C.CEvent)(ptr)
				C.free(unsafe.Pointer(cevent.title))
				C.free(unsafe.Pointer(cevent.location))
				C.free(unsafe.Pointer(cevent.notes))
				C.free(unsafe.Pointer(cevent.startDate))
				C.free(unsafe.Pointer(cevent.endDate))
				C.free(unsafe.Pointer(cevent.duration))
				C.free(unsafe.Pointer(ptr))
			}
			C.free(unsafe.Pointer(result.events))
		}
		if result.error != nil {
			C.free(unsafe.Pointer(result.error))
		}
		C.free(unsafe.Pointer(result))
	}()
	if result.error != nil {
		errorText := C.GoString(result.error)
		return events, errors.New(errorText)
	}

	if count > 0 {
		cevents := (**C.CEvent)(unsafe.Pointer(result.events))
		for i := 0; i < count; i++ {
			cevent := (*C.CEvent)(C.event_at(cevents, C.int(i)))
			s, err := time.Parse("2006-01-02 15:04", C.GoString(cevent.startDate))
			if err != nil {
				return events, err
			}
			e, err := time.Parse("2006-01-02 15:04", C.GoString(cevent.endDate))
			if err != nil {
				return events, err
			}
			d, err := time.ParseDuration(C.GoString(cevent.duration) + "m")
			if err != nil {
				return events, err
			}
			events = append(events, Event{
				Title:     C.GoString(cevent.title),
				Location:  C.GoString(cevent.location),
				Notes:     C.GoString(cevent.notes),
				StartDate: s,
				EndDate:   e,
				Duration:  d,
			})
		}
	}

	return events, nil
}
