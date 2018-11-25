#include <stdio.h>
#include <stdlib.h>
#include <sys/time.h>

struct node{
    int key;
    int val;
    struct node *next;
};
struct table{
    int size;
    struct node **list;
};
struct table *createTable(int size){
    struct table *t = (struct table*)malloc(sizeof(struct table));
    t->size = size;
    t->list = (struct node**)malloc(sizeof(struct node*)*size);
    int i;
    for(i=0;i<size;i++)
        t->list[i] = NULL;
    return t;
}
int hashCode(struct table *t,int key){
    if(key<0)
        return -(key%t->size);
    return key%t->size;
}
void insert(struct table *t,int key,int val){
    int pos = hashCode(t,key);
    struct node *list = t->list[pos];
    struct node *newNode = (struct node*)malloc(sizeof(struct node));
    struct node *temp = list;
    while(temp){
        if(temp->key==key){
            temp->val = val;
            return;
        }
        temp = temp->next;
    }
    newNode->key = key;
    newNode->val = val;
    newNode->next = list;
    t->list[pos] = newNode;
}
int lookup(struct table *t,int key){
    int pos = hashCode(t,key);
    struct node *list = t->list[pos];
    struct node *temp = list;
    while(temp){
        if(temp->key==key){
            return temp->val;
        }
        temp = temp->next;
    }
    return -1;
}

// calculate the time diff between start and end
long delay(struct timeval t1, struct timeval t2)
{
    long d;
    d = (t2.tv_sec - t1.tv_sec) * 1000000;
    d += t2.tv_usec - t1.tv_usec;
    return(d);
}

/* The state word must be initialized to non-zero */
uint32_t myrand(uint32_t r)
{
	/* Algorithm "xor" from p. 4 of Marsaglia, "Xorshift RNGs" */
	r ^= r << 13;
	r ^= r >> 17;
	r ^= r << 5;
	return r;
}

int Sink;

void test(char *name,struct table *t) {
    int mask = (1024*1024)-1;
    struct timeval start, end;
    for(int i=0;i<1000000;i++){
        insert(t,i,i);
    }
    gettimeofday(&start, NULL);
    uint32_t r = start.tv_usec;
    for( int i=0;i<5000000;i++){
        r = myrand(r);
        int index = r & mask;
        insert(t,index,index);
    }
    gettimeofday(&end, NULL);
    printf("%s put = %lf ns/op\n", name,delay(start, end)/(5000000.0/1000));

    gettimeofday(&start, NULL);
    int count=0;
    for( int i=0;i<5000000;i++){
        r = myrand(r);
        int index = r & mask;
        count += lookup(t,index);
    }
    Sink=count;
    gettimeofday(&end, NULL);
    printf("%s get = %lf ns/op\n", name, delay(start, end)/(5000000.0/1000));
}

int main(){
    struct table *t = createTable(256000);

    test("intmap",t);

    t = createTable(1000000);

    test("intmap2",t);
    return 0;
}
